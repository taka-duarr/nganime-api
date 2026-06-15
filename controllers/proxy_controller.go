package controllers

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// ProxyAnimeAPI forwards requests to the Sanka Vollerei Anime API
// This is needed to bypass CORS restrictions when accessing from a browser (Web)
func ProxyAnimeAPI(c *gin.Context) {
	// Get the upstream anime API base URL from env
	animeAPIBase := os.Getenv("ANIME_API_BASE_URL")
	if animeAPIBase == "" {
		animeAPIBase = "https://www.sankavollerei.com/anime"
	}

	// Get the path after /api/proxy/
	proxyPath := c.Param("path")

	// Build the target URL
	targetURL := strings.TrimRight(animeAPIBase, "/") + proxyPath

	// Append query string if any (e.g. ?page=2)
	rawQuery := c.Request.URL.RawQuery
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	// Validate the URL
	parsedURL, err := url.ParseRequestURI(targetURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proxy URL"})
		return
	}

	// Create the outgoing request
	req, err := http.NewRequest(c.Request.Method, parsedURL.String(), c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proxy request"})
		return
	}

	// Forward useful headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NgAnime-Proxy/1.0")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach upstream API", "detail": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read upstream response"})
		return
	}

	// Return the response with the same status code and content type
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
