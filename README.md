
# podcaster

This is a simple podcast generator. It reads all of the metadata from a bunch of MP3 and MP4 (M4A) files, and uses that 
information to build a podcast file in Apple RSS 2.0 format.

For example, suppose you have your M4A files in a subdirectory `mypodcast` of the current directory:

    % podcaster --out mypodcast/index.xml \
      --title "My Cool Podcast" \
      --desc "Assorted audio I've collected" \
      --url http://www.example.com/mypodcast/index.xml
      
    % rsync -av mypodcast/ example.com:/var/www/mypodcast/

You should then find that your feed validates and the enclosures resolve. You can check using https://castfeedvalidator.com 
or any other standard podcast validator.

Note that the program uses the location of the output file and the podcast URL to resolve the absolute URLs of the audio 
files, so you might need to experiment a little or modify the code if your audio file URLs are nowhere near the RSS in URL path terms.
My use case was turning downloaded BBC iPlayer radio shows into a podcast, and it works for that.
(Dear BBC, please just make your Sounds site supply me with podcast feeds.)

The hard work is done by the https://github.com/dhowden/tag and https://github.com/eduncan911/podcast libraries. They 
are licensed BSD 2-clause and MIT, so this is licensed MIT.

meta@pobox.com
2018-12-08
