# crawl

A generic parallel web crawler for Go that provides a simple and flexible API for crawling URLs with configurable concurrency and custom handlers.

## Features

- **Parallel crawling** with configurable worker pools
- **Iterator-based URL generation** using Go 1.23+ iterators
- **Customizable request building** (GET, POST, custom headers, etc.)
- **Flexible response handling** (extract data, save to files, etc.)
- **Optional error handling** via callbacks

## Quick Start

```go
package main

import (
    "context"
    "github.com/gwillem/crawl"
)

func main() {
    ctx := context.Background()

    // Create crawler with default settings
    crawler := crawl.New(ctx, crawl.Config{
        WorkerCount: 5,
    })

    // Simple URL generator
    urls := func(yield func(string) bool) {
        for _, url := range []string{
            "https://example.com",
            "https://www.google.com",
        } {
            if !yield(url) {
                return
            }
        }
    }

    // Run the crawler
    crawler.Run(ctx, urls)
}
```

## Configuration

The `Config` struct provides various options:

```go
type Config struct {
    // WorkerCount is the number of parallel workers. Default: 10.
    WorkerCount int

    // RequestBuilder generates HTTP requests. If nil, uses default GET requests.
    RequestBuilder RequestBuilder

    // ResponseHandler processes responses. If nil, prints status codes.
    ResponseHandler ResponseHandler

    // ErrorHandler handles errors. If nil, errors are ignored.
    ErrorHandler ErrorHandler

    // UserAgent is the User-Agent header. If empty, fetches from API.
    UserAgent string

    // RedirectionPolicy controls how redirects are handled. If nil, uses DefaultRedirectionPolicy.
    RedirectionPolicy RedirectionPolicy

    // Client is the HTTP client to use. If nil, uses http.DefaultClient.
    Client *http.Client
}
```

## Default Helper Functions

### FileURLs

Read URLs from a file (one per line):

```go
crawler.Run(ctx, crawl.FileURLs("urls.txt"))
```

File format:
```
https://example.com
https://www.google.com
# Comments start with #

https://httpbin.org/status/200
```

### ResponseBodySaver

Save response bodies to files:

```go
crawler := crawl.New(ctx, crawl.Config{
    ResponseHandler: crawl.ResponseBodySaver("./output"),
})
```

Files are named by hostname and saved in the specified directory (e.g., `./output/example.com`).

### ErrorLoggerStdout

Log errors to stdout:

```go
crawler := crawl.New(ctx, crawl.Config{
    ErrorHandler: crawl.ErrorLoggerStdout(),
})
```

### Redirection Policies

Control how HTTP redirects are handled:

**DefaultRedirectionPolicy** - Allows a configurable number of redirections:

```go
crawler := crawl.New(ctx, crawl.Config{
    RedirectionPolicy: crawl.DefaultRedirectionPolicy(3), // max 3 redirects
})
```

**SameDomainRedirectionPolicy** - Allows up to 3 redirections, but only within the same domain:

```go
crawler := crawl.New(ctx, crawl.Config{
    RedirectionPolicy: crawl.SameDomainRedirectionPolicy(),
})
```

Example: `example.com` → `www.example.com` is allowed, but `example.com` → `other.com` is blocked.

## Examples

### Custom Request Builder (POST requests)

```go
postBuilder := func(ctx context.Context, url string) (*http.Request, error) {
    body := strings.NewReader(`{"key": "value"}`)
    req, err := http.NewRequestWithContext(ctx, "POST", url, body)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    return req, nil
}

crawler := crawl.New(ctx, crawl.Config{
    RequestBuilder: postBuilder,
})
```

### Custom Response Handler

```go
handler := func(url string, resp *http.Response) error {
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    fmt.Printf("%s -> %d (%d bytes)\n", url, resp.StatusCode, len(body))

    // Process the response body...

    return nil
}

crawler := crawl.New(ctx, crawl.Config{
    ResponseHandler: handler,
})
```

## License

MIT

