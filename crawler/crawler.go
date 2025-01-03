package crawler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type Crawler struct {
	Visited     map[string]bool
	VisitedLock sync.Mutex
	ProductURLs map[string][]string
	Patterns    []*regexp.Regexp
	Client      *http.Client
}

func NewCrawler() *Crawler {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`/product/`),
		regexp.MustCompile(`/products/`),
		regexp.MustCompile(`/productpage`),
		regexp.MustCompile(`/item/`),
		regexp.MustCompile(`/p/`),
		regexp.MustCompile(`/pl/`),
		regexp.MustCompile(`/buy`),
		regexp.MustCompile(`/dp/`),
		regexp.MustCompile(`-p-\d+`),
		regexp.MustCompile(`-p\d+`),
		regexp.MustCompile(`/catalog/product/view/`),
	}
	return &Crawler{
		Visited:     make(map[string]bool),
		ProductURLs: make(map[string][]string),
		Patterns:    patterns,
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout: 10 * time.Second,
				ForceAttemptHTTP2:   false,
			},
		},
	}
}

func (c *Crawler) isProductURL(link string) bool {
	for _, pattern := range c.Patterns {
		if pattern.MatchString(link) {
			return true
		}
	}
	return false
}

func (c *Crawler) fetchURL(urlStr string) (*http.Response, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	return c.Client.Do(req)
}

func (c *Crawler) fetchWithRetry(urlStr string, retries int) (*http.Response, error) {
	for i := 0; i < retries; i++ {
		resp, err := c.fetchURL(urlStr)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		log.Printf("Retrying (%d/%d) for URL: %s\n", i+1, retries, urlStr)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("Failed to fetch URL %s after %d retries", urlStr, retries)
}

func (c *Crawler) Crawl(urlStr, domain string, depth int, maxDepth int) {
	if depth > maxDepth {
		return
	}

	c.VisitedLock.Lock()
	if c.Visited[urlStr] {
		c.VisitedLock.Unlock()
		return
	}
	c.Visited[urlStr] = true
	c.VisitedLock.Unlock()

	resp, err := c.fetchWithRetry(urlStr, 3)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch URL %s: %v\n", urlStr, err)
		return
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Printf("Failed to parse HTML for URL %s: %v\n", urlStr, err)
		return
	}
	log.Println("Crawling items in: ", urlStr)

	c.processLinks(doc, urlStr, domain, depth, maxDepth)
}

func (c *Crawler) processLinks(node *html.Node, base, domain string, depth, maxDepth int) {
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					link := attr.Val
					fullURL := c.resolveURL(base, link)
					if c.isProductURL(fullURL) {
						c.ProductURLs[domain] = append(c.ProductURLs[domain], fullURL)
					} else if strings.Contains(fullURL, domain) {
						go c.Crawl(fullURL, domain, depth+1, maxDepth)
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			f(child)
		}
	}
	f(node)
}

func (c *Crawler) resolveURL(base, href string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	relativeURL, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(relativeURL).String()
}

func (c *Crawler) SaveResults(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %v\n", err)
	}
	defer file.Close()

	data, err := json.MarshalIndent(c.ProductURLs, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize results: %v\n", err)
	}

	file.Write(data)
}
