package main

import (
	"fmt"
	"sync"

	"github.com/Siddhant043/ecommerce-crawler/crawler"
)

func main() {
	crawler := crawler.NewCrawler()
	domains := []string{
		"snitch.co.in",
		"montecarlo.in",
		"www.bewakoof.com/men-t-shirts?manufacturer_brand=bewakoof%C2%AE_bewakoof__air%C2%AE__1.0_bewakoof__heavy__duty%C2%AE__1.0&design=graphic__print_typography_printed", "nykaafashion.com",
		"www.zara.com/in/en/kids-baby-shop-by-size-12-18-months-l7229.html",
		"nykaafashion.com",
		"www.flipkart.com",
		"www.amazon.in",
	}
	maxDepth := 100

	var wg sync.WaitGroup
	for _, domain := range domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			startURL := "https://" + domain
			crawler.Crawl(startURL, domain, 0, maxDepth)
		}(domain)
	}

	wg.Wait()
	crawler.SaveResults("product_urls.json")
	fmt.Println("Crawling completed. Results saved to product_urls.json")
}
