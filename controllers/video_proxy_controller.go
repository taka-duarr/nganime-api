package controllers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// ProxyVideoEmbed fetches a video embed page and strips ad/redirect scripts before serving it.
// Usage: GET /api/video-proxy?url=<encoded-embed-url>
func ProxyVideoEmbed(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url query parameter is required"})
		return
	}

	// Validate URL
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	// Fetch the embed page pretending to be a regular browser
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host))
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch video page", "detail": err.Error()})
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	// If not HTML, just stream it through (e.g. m3u8, mp4, etc.)
	if !strings.Contains(contentType, "text/html") {
		c.Header("Access-Control-Allow-Origin", "*")
		c.DataFromReader(resp.StatusCode, resp.ContentLength, contentType, resp.Body, nil)
		return
	}

	// Read full HTML
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}
	html := string(bodyBytes)

	// ─── Strip Ad Scripts ────────────────────────────────────────────────────

	// 1. Remove <script> blocks that contain known ad/popup patterns
	adPatterns := []string{
		"window.open", "window.top.location", "window.parent.location",
		"top.location", "self.location", "document.location",
		"popunder", "popupunder", "popUnder",
		"adnxs", "googlesyndication", "doubleclick",
		"onclick.*window", "adsbygoogle",
		"exoclick", "trafficjunky", "trafficstars",
		"juicyads", "hilltopads", "propellerads",
	}

	// Remove inline <script>...</script> blocks containing ad keywords
	scriptTagRegex := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptTagRegex.ReplaceAllStringFunc(html, func(match string) string {
		lowerMatch := strings.ToLower(match)
		for _, pattern := range adPatterns {
			if strings.Contains(lowerMatch, strings.ToLower(pattern)) {
				return "<!-- ad script removed -->"
			}
		}
		return match
	})

	// 2. Remove <script src="..."> tags pointing to known ad networks
	adSrcPatterns := []string{
		"googlesyndication", "doubleclick", "adnxs", "exoclick",
		"trafficjunky", "trafficstars", "juicyads", "hilltopads",
		"propellerads", "adsterra", "popads", "popcash",
	}
	externalScriptRegex := regexp.MustCompile(`(?i)<script[^>]+src="([^"]*)"[^>]*>.*?</script>`)
	html = externalScriptRegex.ReplaceAllStringFunc(html, func(match string) string {
		for _, pattern := range adSrcPatterns {
			if strings.Contains(strings.ToLower(match), strings.ToLower(pattern)) {
				return "<!-- external ad script removed -->"
			}
		}
		return match
	})

	// 4. Remove heavily obfuscated scripts (often used by aggressive ad networks)
	// Matches common JS obfuscation patterns like `var _0x1234=` or packed scripts `eval(function(p,a,c,k,e,d))`
	obfuscatedRegex := regexp.MustCompile(`(?is)<script[^>]*>.*?(_0x[0-9a-fA-F]+|eval\(function\(p,a,c,k,e,d\)).*?</script>`)
	html = obfuscatedRegex.ReplaceAllString(html, "<!-- obfuscated ad script removed -->")

	// 5. Block all onclick redirect attempts via injected override script at the top of <body>
	antiRedirectScript := `<script>
// Anti-ad redirect injected by NgAnime proxy
(function() {
  // Block window.open popups
  window.open = function() { return null; };
  // Block top/parent navigation
  try { Object.defineProperty(window, 'top', { get: function() { return window; } }); } catch(e) {}
  try { Object.defineProperty(window, 'parent', { get: function() { return window; } }); } catch(e) {}
  // Intercept all link clicks and block ones leading to other domains
  document.addEventListener('click', function(e) {
    var el = e.target;
    while (el && el.tagName !== 'A') { el = el.parentElement; }
    if (el && el.href) {
      try {
        var linkHost = new URL(el.href).host;
        if (linkHost !== window.location.host) {
          e.preventDefault();
          e.stopPropagation();
          return false;
        }
      } catch(err) {}
    }
  }, true);
})();
</script>`

	// Inject right after <body>
	html = strings.Replace(html, "<body", "<body", 1)
	bodyTagRegex := regexp.MustCompile(`(?i)(<body[^>]*>)`)
	html = bodyTagRegex.ReplaceAllString(html, `$1`+antiRedirectScript)

	// ─── Serve cleaned HTML ──────────────────────────────────────────────────
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Frame-Options", "ALLOWALL")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
