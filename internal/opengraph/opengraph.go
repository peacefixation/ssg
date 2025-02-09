package opengraph

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/peacefixation/static-site-generator/internal/model"
)

var defaultHTTPTimeout = 30 * time.Second
var userAgent = "peacefixation.github.io/static-site-generator:0.0.1" // TODO: add to site.yaml

func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: defaultHTTPTimeout,
	}
}

func Fetch(httpClient *http.Client, links []model.Link) {
	var wg sync.WaitGroup

	for i := range links {
		if i > 0 && i%100 == 0 {
			log.Print("waiting for 1 second before fetching the next 100 links")
			time.Sleep(1 * time.Second)
		}

		wg.Add(1)
		go func(i int) {
			og, err := fetch(httpClient, links[i].URL)
			if err != nil {
				log.Printf("failed to fetch OpenGraph data for %s: %v", links[i].URL, err)
			} else {
				links[i].OpenGraph = og
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func fetch(httpClient *http.Client, url string) (*opengraph.OpenGraph, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	og := opengraph.NewOpenGraph()
	err = og.ProcessHTML(strings.NewReader(string(buf)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenGraph from HTML: %w", err)
	}

	og = fixImageURL(og)

	log.Print(og)

	return og, nil
}
func fixImageURL(og *opengraph.OpenGraph) *opengraph.OpenGraph {
	if len(og.Images) == 0 {
		return og
	}

	// parse the base URL and make sure it has a scheme
	parsedURL, err := url.Parse(og.URL)
	if err != nil {
		return og
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
		og.URL = parsedURL.String()
	}

	// make sure image URLs are absolute
	for i, image := range og.Images {
		// // {"type":"website","url":"3blue1brown.com","title":"3Blue1Brown","description":"Mathematics with a distinct visual perspective. Linear algebra, calculus, neural networks, topology, and more.","determiner":"","site_name":"","locale":"","locales_alternate":null,"images":[{"url":"/favicons/share-thumbnail.jpg","secure_url":"","type":"","width":0,"height":0}],"audios":null,"videos":null}
		// make relative URLs absolute
		// TODO: use url.Parse to handle this
		if !strings.HasPrefix(image.URL, "http") {
			og.Images[i].URL = og.URL + image.URL
		}
	}

	return og
}
