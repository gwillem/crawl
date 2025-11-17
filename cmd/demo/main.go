package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gwillem/crawl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: demo <example>")
		fmt.Println("\nAvailable examples:")
		fmt.Println("  1 - Basic crawl with default handlers")
		fmt.Println("  2 - Read URLs from file")
		fmt.Println("  3 - Custom POST request builder")
		fmt.Println("  4 - Save responses to files")
		fmt.Println("  5 - All features combined")
		os.Exit(1)
	}

	example := os.Args[1]

	switch example {
	case "1":
		basicCrawl()
	case "2":
		fileURLsCrawl()
	case "3":
		customRequestBuilder()
	case "4":
		saveResponses()
	case "5":
		allFeaturesCombined()
	default:
		fmt.Printf("Unknown example: %s\n", example)
		os.Exit(1)
	}
}

// Example 1: Basic crawl with default handlers
func basicCrawl() {
	fmt.Println("=== Example 1: Basic Crawl ===\n")

	ctx := context.Background()

	crawler := crawl.New(ctx, crawl.Config{
		WorkerCount: 3,
	})

	urls := func(yield func(string) bool) {
		for _, url := range []string{
			"https://example.com",
			"https://www.google.com",
			"https://httpbin.org/status/200",
		} {
			if !yield(url) {
				return
			}
		}
	}

	if err := crawler.Run(ctx, urls); err != nil {
		fmt.Fprintf(os.Stderr, "Crawler error: %v\n", err)
	}
}

// Example 2: Read URLs from file
func fileURLsCrawl() {
	fmt.Println("=== Example 2: Read URLs from File ===\n")

	urlsFile := "/tmp/urls.txt"
	sampleURLs := `# Sample URLs file
https://example.com
https://www.google.com

# Another URL
https://httpbin.org/status/200
`
	if err := os.WriteFile(urlsFile, []byte(sampleURLs), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create sample file: %v\n", err)
		return
	}

	fmt.Printf("Created sample file: %s\n\n", urlsFile)

	ctx := context.Background()

	crawler := crawl.New(ctx, crawl.Config{
		WorkerCount:  2,
		ErrorHandler: crawl.ErrorLoggerStdout(),
	})

	if err := crawler.Run(ctx, crawl.FileURLs(urlsFile)); err != nil {
		fmt.Fprintf(os.Stderr, "Crawler error: %v\n", err)
	}
}

// Example 3: Custom POST request builder
func customRequestBuilder() {
	fmt.Println("=== Example 3: Custom POST Request Builder ===\n")

	ctx := context.Background()

	postBuilder := func(ctx context.Context, url string) (*http.Request, error) {
		body := strings.NewReader(`{"key": "value"}`)
		req, err := http.NewRequestWithContext(ctx, "POST", url, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}

	detailedHandler := func(url string, resp *http.Response) error {
		fmt.Printf("%s -> %d %s (Content-Type: %s)\n",
			url,
			resp.StatusCode,
			resp.Status,
			resp.Header.Get("Content-Type"),
		)
		return nil
	}

	crawler := crawl.New(ctx, crawl.Config{
		WorkerCount:     2,
		RequestBuilder:  postBuilder,
		ResponseHandler: detailedHandler,
		ErrorHandler:    crawl.ErrorLoggerStdout(),
	})

	urls := func(yield func(string) bool) {
		for _, url := range []string{
			"https://httpbin.org/post",
			"https://httpbin.org/anything",
		} {
			if !yield(url) {
				return
			}
		}
	}

	if err := crawler.Run(ctx, urls); err != nil {
		fmt.Fprintf(os.Stderr, "Crawler error: %v\n", err)
	}
}

// Example 4: Save responses to files
func saveResponses() {
	fmt.Println("=== Example 4: Save Responses to Files ===\n")

	ctx := context.Background()

	crawler := crawl.New(ctx, crawl.Config{
		WorkerCount:     2,
		ResponseHandler: crawl.ResponseFileDumper("./crawl-output"),
		ErrorHandler:    crawl.ErrorLoggerStdout(),
	})

	urls := func(yield func(string) bool) {
		for _, url := range []string{
			"https://example.com",
			"https://httpbin.org/html",
			"https://httpbin.org/json",
		} {
			if !yield(url) {
				return
			}
		}
	}

	if err := crawler.Run(ctx, urls); err != nil {
		fmt.Fprintf(os.Stderr, "Crawler error: %v\n", err)
	}

	fmt.Println("\nResponses saved to ./crawl-output/")
}

// Example 5: All features combined
func allFeaturesCombined() {
	fmt.Println("=== Example 5: All Features Combined ===\n")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	requestBuilder := func(ctx context.Context, url string) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Custom-Header", "demo-crawler")
		return req, nil
	}

	responseHandler := func(url string, resp *http.Response) error {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		fmt.Printf("%s -> %d %s (%d bytes)\n",
			url,
			resp.StatusCode,
			resp.Status,
			len(body),
		)
		return nil
	}

	crawler := crawl.New(ctx, crawl.Config{
		WorkerCount:     5,
		RequestBuilder:  requestBuilder,
		ResponseHandler: responseHandler,
		ErrorHandler:    crawl.ErrorLoggerStdout(),
	})

	dynamicURLs := func(yield func(string) bool) {
		baseURLs := []string{
			"https://example.com",
			"https://httpbin.org/status/200",
			"https://httpbin.org/status/404",
			"https://httpbin.org/delay/1",
			"https://www.google.com",
		}

		for _, url := range baseURLs {
			if !yield(url) {
				return
			}
		}
	}

	if err := crawler.Run(ctx, dynamicURLs); err != nil {
		fmt.Fprintf(os.Stderr, "Crawler error: %v\n", err)
	}
}
