package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Item struct {
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Author      string `xml:"author"`
}

type Feed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Items   []Item   `xml:"channel>item"`
	Title   string   `xml:"channel>title"`
}

func Parse(r io.Reader) ([]Item, error) {
	var f Feed
	dec := xml.NewDecoder(r)
	err := dec.Decode(&f)
	return f.Items, err
}

func subscribedTo() ([]string, error) {
	var chans []string

	p := os.ExpandEnv("$HOME/.youtube-feed")
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("failed reading %s: %s", p, err)
	}
	defer f.Close()

	scan := bufio.NewScanner(f)

	for scan.Scan() {
		chans = append(chans, strings.TrimSpace(scan.Text()))
	}
	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("failed reading %s: %s", p, err)
	}

	return chans, err
}

const retries = 5

func getLatestVideos(ytChan string, itemChan chan<- Item, status chan<- error) {
	var resp *http.Response
	var err error

	for i := 0; i < retries; i++ {
		resp, err = http.Get("http://gdata.youtube.com/feeds/base/users/" + ytChan + "/uploads?alt=rss&v=2&orderby=published&client=ytapi-youtube-profile")
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				break
			} else {
				resp.Body.Close()
				err = fmt.Errorf("Received status %d, expected %d", resp.StatusCode, http.StatusOK)
			}
		}
	}

	if err != nil {
		status <- fmt.Errorf("Failed getting channel feed for %s: %s", ytChan, err)
		return
	}
	defer resp.Body.Close()

	items, err := Parse(resp.Body)
	if err != nil {
		err = fmt.Errorf("Failed parsing feed of %s: %s", ytChan, err)
	}

	for _, it := range items {
		itemChan <- it
	}

	status <- err
}

func main() {
	subs, err := subscribedTo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed reading channel list: %s\n", err)
		os.Exit(1)
	}

	itemChan := make(chan Item)
	status := make(chan error)

	var items []Item
	go func() {
		for it := range itemChan {
			it.Title = fmt.Sprintf("[%s] %s", it.Author, it.Title)
			items = append(items, it)
		}
	}()

	for _, ytChan := range subs {
		go getLatestVideos(ytChan, itemChan, status)
	}

	for _ = range subs {
		if err := <-status; err != nil {
			fmt.Fprintln(os.Stderr, "Warning:", err)
		}
	}
	close(itemChan)

	feed := Feed{Title: "Combined YouTube Feed", Items: items, Version: "2.0"}
	enc := xml.NewEncoder(os.Stdout)
	if err := enc.Encode(feed); err != nil {
		fmt.Fprintf(os.Stderr, "Failed writing feed: %s\n", err)
		os.Exit(1)
	}
}
