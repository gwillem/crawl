package crawl

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
func NoopErrorHandler(url string, err error) {
}

// FileURLs reads URLs from a file (one per line) and returns an iterator.
// Empty lines and lines starting with # are skipped.
func FileURLs(path string) iter.Seq[string] {
	return func(yield func(string) bool) {
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", path, err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if line == "" || strings.HasPrefix(line, "#") {
				continue
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

// ResponseFileDumper returns a ResponseHandler that saves response bodies to files.
// Files are named using a hash of the URL and saved in the specified directory.
// If dir is empty, uses "./responses" as the default directory.
func ResponseFileDumper(dir string) ResponseHandler {
	if dir == "" {
		dir = "./responses"
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dir, err)
	}

	return func(url string, resp *http.Response) error {
		hash := sha256.Sum256([]byte(url))
		filename := hex.EncodeToString(hash[:8]) + ".html"
		filepath := filepath.Join(dir, filename)

		file, err := os.Create(filepath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filepath, err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write response to %s: %w", filepath, err)
		}

		fmt.Printf("%s -> %d %s (saved to %s)\n", url, resp.StatusCode, resp.Status, filepath)
		return nil
	}
}

// ErrorLoggerStdout returns an ErrorHandler that logs errors to stdout.
func ErrorLoggerStdout() ErrorHandler {
	return func(url string, err error) {
		fmt.Fprintf(os.Stderr, "ERROR: %s -> %v\n", url, err)
	}
}
