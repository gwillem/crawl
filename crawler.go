package crawl

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
)

const defaultMaxRedirects = 3

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
	if config.RedirectionPolicy == nil {
		config.RedirectionPolicy = DefaultRedirectionPolicy(defaultMaxRedirects)
	}

	client := config.Client
	if client == nil {
		client = &http.Client{}
	}

	clientCopy := *client
	if clientCopy.CheckRedirect == nil {
		clientCopy.CheckRedirect = config.RedirectionPolicy
	}

	if clientCopy.Transport == nil {
		clientCopy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	} else if transport, ok := clientCopy.Transport.(*http.Transport); ok {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	client = &clientCopy

	userAgent := getUserAgent(ctx, config)
	secChUa := generateSecChUa(userAgent)

	return &Crawler{
		config:    config,
		userAgent: userAgent,
		secChUa:   secChUa,
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

	// Set Chrome-like headers if not already set by RequestBuilder
	setHeaderIfNotExists := func(key, value string) {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}

	// Always override User-Agent with our configured one
	req.Header.Set("User-Agent", c.userAgent)

	// Standard Chrome headers
	setHeaderIfNotExists("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	setHeaderIfNotExists("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8,nl;q=0.7,sv;q=0.6")
	setHeaderIfNotExists("Cache-Control", "no-cache")
	setHeaderIfNotExists("Pragma", "no-cache")
	setHeaderIfNotExists("Priority", "u=0, i")
	setHeaderIfNotExists("Referer", "https://www.google.com/")
	setHeaderIfNotExists("Sec-Ch-Ua", c.secChUa)
	setHeaderIfNotExists("Sec-Ch-Ua-Mobile", "?0")
	setHeaderIfNotExists("Sec-Ch-Ua-Platform", `"macOS"`)
	setHeaderIfNotExists("Sec-Fetch-Dest", "document")
	setHeaderIfNotExists("Sec-Fetch-Mode", "navigate")
	setHeaderIfNotExists("Sec-Fetch-Site", "same-origin")
	setHeaderIfNotExists("Sec-Fetch-User", "?1")
	setHeaderIfNotExists("Upgrade-Insecure-Requests", "1")

	resp, err := c.client.Do(req)
	if err != nil {
		c.config.ErrorHandler(url, err)
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				c.config.ErrorHandler(url, err)
			}
		}
	}()

	if err := c.config.ResponseHandler(url, resp); err != nil {
		c.config.ErrorHandler(url, err)
	}
}
