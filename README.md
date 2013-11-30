youtube-feed
============

Combine multiple YouTube user feeds into a single RSS-Feed.

Compiling
---------

`go get github.com/kch42/youtube-feed`

Usage
-----

Put all the usernames of YouTubers you want to receive updates from into the file `~/.youtube-feed`.
`youtube-feed` will now output the combined feed on `stdout`. You can use youtube-feed with [newsbeuter](http://www.newsbeuter.org). Just put `exec:youtube-feed` into your `urls` file.
