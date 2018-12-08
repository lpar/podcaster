package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eduncan911/podcast"

	"github.com/dhowden/tag"
	"github.com/spf13/pflag"
)

var pc_url *string = pflag.StringP("url", "u", "", "URL of feed file including filename")
var outfile *string = pflag.StringP("out", "o", "index.xml", "output file")
var pc_title *string = pflag.StringP("title", "t", "", "podcast title")
var pc_desc *string = pflag.StringP("desc", "d", "", "podcast description")
var helpreq *bool = pflag.BoolP("help", "h", false, "request help")

type Podcast struct {
	episodes    []*Episode
	BaseDir     string
	BaseURL     *url.URL
	Title       string
	Description string
	lastshow    string
	multishow   bool
	OutFile     string
}

// Struct used to store info extracted from the file metadata
type Episode struct {
	Title         string    // track title
	Show          string    // album
	Provider      string    // artist
	Description   string    // comment
	Episode       int       // track number
	Series        int       // disc number
	Updated       time.Time // file mod time
	URL           *url.URL  // computed from path
	Bytes         int64
	EnclosureType podcast.EnclosureType
}

func (e *Episode) Dump() {
	fmt.Printf("%s s%d e%d: %s\n", e.Show, e.Series, e.Episode, e.Title)
}

//// Implement sort.Interface on Podcast

// Len returns the number of Episodes in the Podcast
func (p *Podcast) Len() int {
	return len(p.episodes)
}

// Less returns whether episode i should sort before episode j.
// We sort by Provider, reverse Show, reverse Series, reverse Episode, reverse Title.
// Show is sorted in reverse so "The Mark Steel Lecture Series 3" comes before
// "The Mark Steel Lecture Series 2", for example.
func (p *Podcast) Less(i, j int) bool {
	a := p.episodes[i]
	b := p.episodes[j]
	// Is a < b?
	if a.Provider < b.Provider {
		return true
	}
	if a.Provider > b.Provider {
		return false
	}
	if a.Show < b.Show {
		return !true
	}
	if a.Show > b.Show {
		return !false
	}
	// For reduced confusion, use ! on the return value for reversed sort
	// but keep everything else the same.
	if a.Series < b.Series {
		return !true
	}
	if a.Series > b.Series {
		return !false
	}
	if a.Episode < b.Episode {
		return !true
	}
	if a.Episode > b.Episode {
		return !false
	}
	if a.Title < b.Title {
		return !true
	}
	if a.Title > b.Title {
		return !false
	}
	return false
}

// Swap swaps two episodes in the podcast
func (p *Podcast) Swap(i, j int) {
	tep := p.episodes[i]
	p.episodes[i] = p.episodes[j]
	p.episodes[j] = tep
}

////

func (p *Podcast) Add(path string, tags tag.Metadata, info os.FileInfo, ext string) error {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("can't resolve absolute path of file %s: %v", path, err)
	}
	fpath, err := filepath.Rel(p.BaseDir, abspath)
	if err != nil {
		return fmt.Errorf("can't resolve podcast file %s relative to output directory %s: %v", abspath, p.BaseDir, err)
	}
	furl, err := url.Parse(fpath)
	if err != nil {
		return fmt.Errorf("can't parse path %s as URL: %v", path, err)
	}
	eurl := p.BaseURL.ResolveReference(furl)
	t, _ := tags.Track()
	d, _ := tags.Disc()
	enctype := podcast.M4A
	if ext == ".mp3" {
		enctype = podcast.MP3
	}
	ep := &Episode{
		Title:         tags.Title(),
		Show:          tags.Album(),
		Provider:      tags.Artist(),
		Description:   tags.Comment(),
		Episode:       t,
		Series:        d,
		Updated:       info.ModTime(),
		Bytes:         info.Size(),
		EnclosureType: enctype,
		URL:           eurl,
	}
	// Check whether this is going to be a multi-show podcast
	if p.lastshow != "" && p.lastshow != ep.Show {
		p.multishow = true
	}
	if p.lastshow == "" {
		p.lastshow = ep.Show
	}
	p.episodes = append(p.episodes, ep)
	return nil
}

func (p *Podcast) Walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".m4a" || ext == ".mp3" {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("can't open %s to read tags: %v", path, err)
		}
		tags, err := tag.ReadFrom(f)
		if err != nil {
			return fmt.Errorf("can't parse tags from %s: %v", path, err)
		}
		p.Add(path, tags, info, ext)
	}
	return nil
}

func (p *Podcast) Sort() {
	sort.Sort(p)
}

func (p *Podcast) Dump() {
	fmt.Printf("%s\n%s\n", p.BaseURL, p.BaseDir)
	for _, e := range p.episodes {
		e.Dump()
	}
}

func (p *Podcast) Write() error {
	now := time.Now()
	pc := podcast.New(p.Title, p.BaseURL.String(), p.Description, &now, &now)
	for _, ep := range p.episodes {
		// Add a title to the podcast feed based on the show name, if no title was provided
		if p.Title == "" && ep.Show != "" {
			p.Title = ep.Show
		}
		title := ep.Title
		if p.multishow {
			title = ep.Show + ": " + ep.Title
		}
		item := podcast.Item{
			Title:       title,
			Link:        ep.URL.String(),
			Description: ep.Description,
			PubDate:     &ep.Updated,
			IOrder:      fmt.Sprintf("%d", 100*ep.Series+ep.Episode),
		}
		item.AddEnclosure(ep.URL.String(), ep.EnclosureType, ep.Bytes)
		_, err := pc.AddItem(item)
		if err != nil {
			return fmt.Errorf("error adding item %s: %v", ep.URL.String(), err)
		}
	}
	outfp, err := os.Create(p.OutFile)
	if err != nil {
		return fmt.Errorf("can't open %s for writing: %v", outfile, err)
	}
	err = pc.Encode(outfp)
	return err
}

func main() {
	pflag.Parse()

	if *helpreq {
		pflag.Usage()
		return
	}

	if len(pflag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "you must specify one or more files or directories to index (--help for help)")
		return
	}

	if pc_url == nil || *pc_url == "" {
		fmt.Fprintf(os.Stderr, "missing argument --url to specify podcast feed URL")
		return
	}
	baseurl, err := url.Parse(*pc_url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid feed URL %s: %v", *pc_url, err)
	}

	absoutfile, err := filepath.Abs(*outfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't resolve base directory of output file %s: %v", *outfile, err)
	}

	podcast := &Podcast{
		BaseDir: filepath.Dir(absoutfile),
		BaseURL: baseurl,
		Title:   *pc_title,
		OutFile: *outfile,
	}

	for _, fname := range pflag.Args() {
		filepath.Walk(fname, podcast.Walk)
	}
	podcast.Sort()
	err = podcast.Write()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing podcast: %v", err)
	}
}
