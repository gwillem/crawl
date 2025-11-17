package crawl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// DefaultRequestBuilder creates a simple GET request for the given URL.
func DefaultRequestBuilder(ctx context.Context, url string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, "GET", url, nil)
}

// DefaultResponseHandler prints the status code of the response.
func DefaultResponseHandler(url string, resp *http.Response) error {
	fmt.Printf("%s -> %d %s\n", url, resp.StatusCode, resp.Status)
	return nil
}

// NoopErrorHandler is a no-op error handler that ignores all errors.
func NoopErrorHandler(_ string, _ error) {
}

// FileURLs reads hostnames or URLs from a file (one per line) and returns an iterator.
// Empty lines and lines starting with # are skipped.
// If a line doesn't have a scheme (http:// or https://), "https://" is automatically prefixed.
func FileURLs(path string) iter.Seq[string] {
	return func(yield func(string) bool) {
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", path, err)
			return
		}
		defer file.Close() //nolint:errcheck

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			if !strings.HasPrefix(line, "http://") && !strings.HasPrefix(line, "https://") {
				line = "https://" + line
			}

			if !yield(line) {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", path, err)
		}
	}
}

// ResponseBodySaver returns a ResponseHandler that saves response bodies to files.
// Files are named by hostname: <dir>/<hostname>
// If dir is empty, uses "snapshot" as the default directory.
func ResponseBodySaver(dir string) ResponseHandler {
	if dir == "" {
		dir = "snapshot"
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dir, err)
	}

	return func(urlStr string, resp *http.Response) error {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return fmt.Errorf("failed to parse URL %s: %w", urlStr, err)
		}

		filename := parsedURL.Host
		filepath := filepath.Join(dir, filename)

		file, err := os.Create(filepath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filepath, err)
		}
		defer file.Close() //nolint:errcheck

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write response to %s: %w", filepath, err)
		}

		fmt.Printf("%d %s\n", resp.StatusCode, parsedURL.Host)
		return nil
	}
}

// ErrorLoggerStdout returns an ErrorHandler that logs errors to stdout.
func ErrorLoggerStdout() ErrorHandler {
	return func(url string, err error) {
		fmt.Fprintf(os.Stderr, "ERROR: %s -> %v\n", url, err)
	}
}

// DefaultRedirectionPolicy allows up to n redirections.
func DefaultRedirectionPolicy(maxRedirects int) RedirectionPolicy {
	return func(_ *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

// SameDomainRedirectionPolicy allows up to 3 redirections but only if they share the same public suffix.
// For example, example.com and www.example.com share the same domain, but example.com and other.com do not.
func SameDomainRedirectionPolicy() RedirectionPolicy {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return http.ErrUseLastResponse
		}

		if len(via) == 0 {
			return nil
		}

		originalDomain, err := publicsuffix.EffectiveTLDPlusOne(via[0].URL.Host)
		if err != nil {
			return http.ErrUseLastResponse
		}

		currentDomain, err := publicsuffix.EffectiveTLDPlusOne(req.URL.Host)
		if err != nil {
			return http.ErrUseLastResponse
		}

		if originalDomain != currentDomain {
			return http.ErrUseLastResponse
		}

		return nil
	}
}
