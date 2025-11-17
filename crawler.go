package crawl

import (
	"context"
	"net/http"
	"sync"
)

// New creates a new Crawler with the given configuration.
// It fetches the user agent from the API if not provided in the config.
func New(ctx context.Context, config Config) *Crawler {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 10
	}

	if config.RequestBuilder == nil {
		config.RequestBuilder = DefaultRequestBuilder
	}
	if config.ResponseHandler == nil {
		config.ResponseHandler = DefaultResponseHandler
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = NoopErrorHandler
	}

	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}

	userAgent := getUserAgent(ctx, config)

	return &Crawler{
		config:    config,
		userAgent: userAgent,
		client:    client,
	}
}

// Run starts crawling URLs from the generator with N parallel workers.
func (c *Crawler) Run(ctx context.Context, urlGen URLGenerator) error {
	urlChan := make(chan string, c.config.WorkerCount*2)
	var wg sync.WaitGroup

	for i := 0; i < c.config.WorkerCount; i++ {
		wg.Add(1)
		go c.worker(ctx, urlChan, &wg)
	}

	go func() {
		defer close(urlChan)
		for url := range urlGen {
			select {
			case <-ctx.Done():
				return
			case urlChan <- url:
			}
		}
	}()

	wg.Wait()
	return ctx.Err()
}

// worker processes URLs from the channel.
func (c *Crawler) worker(ctx context.Context, urlChan <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case url, ok := <-urlChan:
			if !ok {
				return
			}
			c.processURL(ctx, url)
		}
	}
}

// processURL handles a single URL: builds request, sends it, and handles response.
func (c *Crawler) processURL(ctx context.Context, url string) {
	req, err := c.config.RequestBuilder(ctx, url)
	if err != nil {
		c.config.ErrorHandler(url, err)
		return
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		c.config.ErrorHandler(url, err)
		return
	}
	defer resp.Body.Close()

	if err := c.config.ResponseHandler(url, resp); err != nil {
		c.config.ErrorHandler(url, err)
	}
}
