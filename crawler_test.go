package crawl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ctx := context.Background()

	t.Run("default config", func(t *testing.T) {
		crawler := New(ctx, Config{})

		if crawler == nil {
			t.Fatal("expected crawler to be created")
		}
		if crawler.config.WorkerCount != 10 {
			t.Errorf("expected default worker count 10, got %d", crawler.config.WorkerCount)
		}
		if crawler.config.RequestBuilder == nil {
			t.Error("expected default request builder")
		}
		if crawler.config.ResponseHandler == nil {
			t.Error("expected default response handler")
		}
	})

	t.Run("custom config", func(t *testing.T) {
		customBuilder := func(ctx context.Context, url string) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, "POST", url, nil)
		}

		crawler := New(ctx, Config{
			WorkerCount:    5,
			RequestBuilder: customBuilder,
		})

		if crawler.config.WorkerCount != 5 {
			t.Errorf("expected worker count 5, got %d", crawler.config.WorkerCount)
		}
	})
}

func TestCrawlerRun(t *testing.T) {
	t.Run("basic crawl", func(t *testing.T) {
		// Create test server
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		}))
		defer server.Close()

		ctx := context.Background()

		// Track handled URLs
		var handledCount atomic.Int32
		handler := func(url string, resp *http.Response) error {
			handledCount.Add(1)
			return nil
		}

		crawler := New(ctx, Config{
			WorkerCount:     3,
			ResponseHandler: handler,
		})

		// Generate URLs
		urls := func(yield func(string) bool) {
			for i := 0; i < 5; i++ {
				if !yield(server.URL) {
					return
				}
			}
		}

		// Run crawler
		err := crawler.Run(ctx, urls)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if handledCount.Load() != 5 {
			t.Errorf("expected 5 handled URLs, got %d", handledCount.Load())
		}
		if requestCount.Load() != 5 {
			t.Errorf("expected 5 requests, got %d", requestCount.Load())
		}
	})

	t.Run("with context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())

		crawler := New(ctx, Config{
			WorkerCount: 2,
		})

		// Generate many URLs
		urls := func(yield func(string) bool) {
			for i := 0; i < 100; i++ {
				if !yield(server.URL) {
					return
				}
			}
		}

		// Cancel after 50ms
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := crawler.Run(ctx, urls)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
	})

	t.Run("error handling", func(t *testing.T) {
		ctx := context.Background()

		var errorCount atomic.Int32
		errorHandler := func(url string, err error) {
			errorCount.Add(1)
		}

		crawler := New(ctx, Config{
			WorkerCount:  2,
			ErrorHandler: errorHandler,
		})

		// Generate invalid URLs
		urls := func(yield func(string) bool) {
			for _, url := range []string{
				"http://invalid-domain-that-does-not-exist-12345.com",
				"http://another-invalid-domain-99999.com",
			} {
				if !yield(url) {
					return
				}
			}
		}

		crawler.Run(ctx, urls)

		if errorCount.Load() != 2 {
			t.Errorf("expected 2 errors, got %d", errorCount.Load())
		}
	})
}

func TestDefaultRequestBuilder(t *testing.T) {
	ctx := context.Background()
	url := "https://example.com"

	req, err := DefaultRequestBuilder(ctx, url)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("expected GET method, got %s", req.Method)
	}
	if req.URL.String() != url {
		t.Errorf("expected URL %s, got %s", url, req.URL.String())
	}
}

func TestDefaultResponseHandler(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
	}

	err := DefaultResponseHandler("https://example.com", resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	output := string(out)
	if !strings.Contains(output, "https://example.com") {
		t.Errorf("expected output to contain URL")
	}
	if !strings.Contains(output, "200") {
		t.Errorf("expected output to contain status code")
	}
}

func TestFileURLs(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "urls.txt")

	content := `https://example.com
# This is a comment
https://google.com

https://httpbin.org/status/200
`
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Read URLs
	var urls []string
	for url := range FileURLs(tmpFile) {
		urls = append(urls, url)
	}

	expected := []string{
		"https://example.com",
		"https://google.com",
		"https://httpbin.org/status/200",
	}

	if len(urls) != len(expected) {
		t.Errorf("expected %d URLs, got %d", len(expected), len(urls))
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("expected URL %s at index %d, got %s", expected[i], i, url)
		}
	}
}

func TestResponseBodySaver(t *testing.T) {
	tmpDir := t.TempDir()
	handler := ResponseBodySaver(tmpDir)

	body := io.NopCloser(strings.NewReader("test content"))
	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       body,
	}

	url := "https://example.com/test"
	err := handler(url, resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedFilename := "example.com"
	expectedPath := filepath.Join(tmpDir, expectedFilename)

	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", expectedPath, err)
	}

	if string(content) != "test content" {
		t.Errorf("expected content 'test content', got '%s'", string(content))
	}
}

func TestErrorLoggerStdout(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	handler := ErrorLoggerStdout()
	handler("https://example.com", fmt.Errorf("test error"))

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stderr = oldStderr

	output := string(out)
	if !strings.Contains(output, "ERROR") {
		t.Errorf("expected output to contain 'ERROR'")
	}
	if !strings.Contains(output, "https://example.com") {
		t.Errorf("expected output to contain URL")
	}
	if !strings.Contains(output, "test error") {
		t.Errorf("expected output to contain error message")
	}
}

func TestUserAgent(t *testing.T) {
	ctx := context.Background()

	t.Run("uses provided user agent", func(t *testing.T) {
		customUA := "Custom User Agent"
		crawler := New(ctx, Config{
			UserAgent: customUA,
		})

		if crawler.userAgent != customUA {
			t.Errorf("expected user agent '%s', got '%s'", customUA, crawler.userAgent)
		}
	})

	t.Run("fetches user agent if not provided", func(t *testing.T) {
		crawler := New(ctx, Config{})

		if crawler.userAgent == "" {
			t.Error("expected user agent to be set")
		}
	})
}

func TestDefaultRedirectionPolicy(t *testing.T) {
	policy := DefaultRedirectionPolicy(3)

	t.Run("allows up to 3 redirects", func(t *testing.T) {
		req := &http.Request{URL: &url.URL{Host: "example.com"}}
		via := make([]*http.Request, 2)
		err := policy(req, via)
		if err != nil {
			t.Errorf("expected nil error for 2 redirects, got %v", err)
		}
	})

	t.Run("stops at 3 redirects", func(t *testing.T) {
		req := &http.Request{URL: &url.URL{Host: "example.com"}}
		via := make([]*http.Request, 3)
		err := policy(req, via)
		if err != http.ErrUseLastResponse {
			t.Errorf("expected ErrUseLastResponse for 3 redirects, got %v", err)
		}
	})
}

func TestSameDomainRedirectionPolicy(t *testing.T) {
	policy := SameDomainRedirectionPolicy()

	t.Run("allows same domain redirects", func(t *testing.T) {
		via := []*http.Request{
			{URL: &url.URL{Host: "example.com"}},
		}
		req := &http.Request{URL: &url.URL{Host: "www.example.com"}}
		err := policy(req, via)
		if err != nil {
			t.Errorf("expected nil error for same domain redirect, got %v", err)
		}
	})

	t.Run("blocks cross-domain redirects", func(t *testing.T) {
		via := []*http.Request{
			{URL: &url.URL{Host: "example.com"}},
		}
		req := &http.Request{URL: &url.URL{Host: "other.com"}}
		err := policy(req, via)
		if err != http.ErrUseLastResponse {
			t.Errorf("expected ErrUseLastResponse for cross-domain redirect, got %v", err)
		}
	})

	t.Run("stops at 3 redirects even for same domain", func(t *testing.T) {
		via := []*http.Request{
			{URL: &url.URL{Host: "example.com"}},
			{URL: &url.URL{Host: "www.example.com"}},
			{URL: &url.URL{Host: "api.example.com"}},
		}
		req := &http.Request{URL: &url.URL{Host: "example.com"}}
		err := policy(req, via)
		if err != http.ErrUseLastResponse {
			t.Errorf("expected ErrUseLastResponse for 3 redirects, got %v", err)
		}
	})
}

func TestSecChUaGeneration(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "Chrome 144",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
			expected:  `"Chromium";v="144", "Google Chrome";v="144", "Not_A Brand";v="99"`,
		},
		{
			name:      "Chrome 142",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.5.6789.123 Safari/537.36",
			expected:  `"Chromium";v="142", "Google Chrome";v="142", "Not_A Brand";v="99"`,
		},
		{
			name:      "No version fallback",
			userAgent: "Some Random User Agent",
			expected:  `"Chromium";v="144", "Google Chrome";v="144", "Not_A Brand";v="99"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSecChUa(tt.userAgent)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
