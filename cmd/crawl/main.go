package main

import (
	"fmt"
	"regexp"

	"github.com/fatih/color"
	"github.com/gocolly/colly"
)

func main() {

	ctr := 0

	// Instantiate default collector
	c := colly.NewCollector(
		// Visit only domains: hackerspaces.org, wiki.hackerspaces.org
		// colly.AllowedDomains("localhost"),
		//colly.MaxDepth(1),
		colly.URLFilters(
			// regexp.MustCompile("http://httpbin\\.org/(|e.+)$"),
			regexp.MustCompile("http://localhost.+"),
		),
	)

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Print link
		//fmt.Printf("Link found: %q -> %s\n", e.Text, link)

		// Visit link found on page
		// Only those links are visited which are in AllowedDomains
		c.Visit(e.Request.AbsoluteURL(link))
	})

	c.OnError(func(r *colly.Response, err error) {
		str := fmt.Sprintf("\nRequest URL: %s ==> Error: %s (%d)\n", r.Request.URL, err, r.StatusCode)
		color.Red(str)
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		ctr++
		if ctr < 100 || ctr%1000 == 0 {
			fmt.Printf("\nVisiting: %s", r.URL.String())
		} else {
			fmt.Printf(".")
		}
	})
	// Start scraping on https://hackerspaces.org
	c.Visit("http://localhost:8143")
}
