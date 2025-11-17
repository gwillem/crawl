package crawl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Default user agent fallback if API is unavailable
const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"

// fetchUserAgent retrieves the latest user agent from the Sansec API.
// Returns the default user agent if the API call fails.
func fetchUserAgent(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.sansec.io/v1/useragent/latest", nil)
	if err != nil {
		return defaultUserAgent
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return defaultUserAgent
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return defaultUserAgent
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return defaultUserAgent
	}

	userAgent := strings.TrimSpace(string(body))
	if userAgent == "" {
		return defaultUserAgent
	}

	return userAgent
}

// getUserAgent returns the user agent to use for the crawler.
// If a user agent is provided in the config, it uses that.
// Otherwise, it fetches the latest from the API.
func getUserAgent(ctx context.Context, config Config) string {
	if config.UserAgent != "" {
		return config.UserAgent
	}
	return fetchUserAgent(ctx)
}

// extractChromeVersion extracts the Chrome version from a user agent string.
// Example: "Chrome/144.0.0.0" -> "144"
func extractChromeVersion(userAgent string) string {
	re := regexp.MustCompile(`Chrome/(\d+)\.`)
	matches := re.FindStringSubmatch(userAgent)
	if len(matches) > 1 {
		return matches[1]
	}
	return "144" // Default fallback
}

// generateSecChUa generates the Sec-Ch-Ua header from the user agent.
// Example: Chrome/144.0.0.0 -> "Chromium";v="144", "Google Chrome";v="144", "Not_A Brand";v="99"
func generateSecChUa(userAgent string) string {
	version := extractChromeVersion(userAgent)
	return fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not_A Brand";v="99"`, version, version)
}
