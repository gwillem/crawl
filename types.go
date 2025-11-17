// Package crawl provides a generic parallel web crawler.
package crawl

import (
	"context"
	"iter"
	"net/http"
)

// URLGenerator is a function that yields URLs to crawl using Go 1.23+ iterators.
type URLGenerator = iter.Seq[string]

// RequestBuilder is an optional callback that generates HTTP requests for a given URL.
// If nil, a default GET request will be used.
type RequestBuilder func(ctx context.Context, url string) (*http.Request, error)

// ResponseHandler is an optional callback that processes HTTP responses.
// If nil, a default handler that prints the status code will be used.
type ResponseHandler func(url string, resp *http.Response) error

// ErrorHandler is an optional callback that handles errors during crawling.
// If nil, errors will be silently ignored.
type ErrorHandler func(url string, err error)

// RedirectionPolicy is a function that determines whether to follow a redirect.
// It receives the current request and a slice of all previous requests (via).
// Return an error to stop following redirects, or nil to continue.
type RedirectionPolicy func(req *http.Request, via []*http.Request) error

// Config contains configuration options for the crawler.
type Config struct {
	// WorkerCount is the number of parallel workers. Default: 10.
	WorkerCount int

	// RequestBuilder generates HTTP requests. If nil, uses default GET requests.
	RequestBuilder RequestBuilder

	// ResponseHandler processes responses. If nil, prints status codes.
	ResponseHandler ResponseHandler

	// ErrorHandler handles errors. If nil, errors are ignored.
	ErrorHandler ErrorHandler

	// UserAgent is the User-Agent header to use. If empty, fetches from API.
	UserAgent string

	// RedirectionPolicy controls how redirects are handled. If nil, uses DefaultRedirectionPolicy.
	RedirectionPolicy RedirectionPolicy

	// Client is the HTTP client to use. If nil, uses http.DefaultClient.
	Client *http.Client
}

// Crawler represents a web crawler instance.
type Crawler struct {
	config    Config
	userAgent string
	secChUa   string
	client    *http.Client
}
