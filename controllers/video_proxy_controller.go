package controllers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// unpackDictionary extracts the dictionary from a Dean Edwards packed script.
// The packed format is: eval(function(p,a,c,k,e,d){...}('code',base,count,'dict'.split('|')))
func unpackDictionary(html string) (packedCode string, items []string, base int, ok bool) {
	// Find the eval(function(p,a,c,k,e,d) block
	evalRe := regexp.MustCompile(`eval\(function\(p,a,c,k,e,d\).*?'([^']+)'\.split\('\|'\)\)`)
	evalMatch := evalRe.FindStringSubmatch(html)
	if len(evalMatch) < 2 {
		return "", nil, 0, false
	}
	items = strings.Split(evalMatch[1], "|")

	// Extract the packed code and base
	codeRe := regexp.MustCompile(`\}\('(.*?)',(\d+),(\d+),'`)
	codeMatch := codeRe.FindStringSubmatch(html)
	if len(codeMatch) < 3 {
		return "", nil, 0, false
	}
	packedCode = codeMatch[1]
	base, _ = strconv.Atoi(codeMatch[2])
	return packedCode, items, base, true
}

// decodeWord converts a base-N encoded word back to its dictionary index
func decodeWord(word string, base int) (int, bool) {
	n := 0
	for _, c := range word {
		var d int
		if c >= '0' && c <= '9' {
			d = int(c - '0')
		} else if c >= 'a' && c <= 'z' {
			d = int(c-'a') + 10
		} else if c >= 'A' && c <= 'Z' {
			d = int(c-'A') + 36
		} else {
			return 0, false
		}
		if d >= base {
			return 0, false
		}
		n = n*base + d
	}
	return n, true
}

// decodePacked decodes a Dean Edwards packed script using its dictionary
func decodePacked(code string, items []string, base int) string {
	wordRe := regexp.MustCompile(`\b\w+\b`)
	return wordRe.ReplaceAllStringFunc(code, func(word string) string {
		idx, ok := decodeWord(word, base)
		if ok && idx < len(items) && items[idx] != "" {
			return items[idx]
		}
		return word
	})
}

// extractM3U8FromEmbed fetches an embed page and extracts the m3u8 stream URL
func extractM3U8FromEmbed(embedURL string) (string, error) {
	parsedURL, err := url.ParseRequestURI(embedURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	req, err := http.NewRequest("GET", embedURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host))
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch embed page: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}
	html := string(bodyBytes)

	// Try to find packed script and decode it
	packedCode, items, base, ok := unpackDictionary(html)
	if ok {
		decoded := decodePacked(packedCode, items, base)

		// Try to find hls2 URL (full URL with token, usually the best)
		hls2Re := regexp.MustCompile(`"hls2":"(https?://[^"]+\.m3u8[^"]*)"`)
		if m := hls2Re.FindStringSubmatch(decoded); len(m) > 1 {
			return m[1], nil
		}

		// Try hls3
		hls3Re := regexp.MustCompile(`"hls3":"(https?://[^"]+)"`)
		if m := hls3Re.FindStringSubmatch(decoded); len(m) > 1 {
			return m[1], nil
		}

		// Try any URL ending in .m3u8
		m3u8Re := regexp.MustCompile(`(https?://[^\s"']+\.m3u8[^\s"']*)`)
		if m := m3u8Re.FindStringSubmatch(decoded); len(m) > 1 {
			return m[1], nil
		}
	}

	// Fallback: try to find m3u8 URL directly in the raw HTML
	directRe := regexp.MustCompile(`(https?://[^\s"']+\.m3u8[^\s"']*)`)
	if m := directRe.FindStringSubmatch(html); len(m) > 1 {
		return m[1], nil
	}

	return "", fmt.Errorf("could not find m3u8 stream URL in embed page")
}

// cleanPlayerHTML generates a minimal, ad-free video player page using HLS.js
func cleanPlayerHTML(m3u8URL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>NgAnime Player</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
html,body{width:100%%;height:100%%;background:#000;overflow:hidden;color:white;font-family:sans-serif}
video{width:100%%;height:100%%;object-fit:contain;background:#000}
#debug{position:absolute;top:0;left:0;z-index:9999;background:rgba(0,0,0,0.8);padding:10px;font-size:12px;width:100%%;height:30%%;overflow-y:auto;pointer-events:none;}
</style>
</head>
<body>
<div id="debug">Initializing...</div>
<video id="video" controls autoplay playsinline></video>
<script src="https://cdn.jsdelivr.net/npm/hls.js@1.5.17/dist/hls.min.js"></script>
<script>
(function(){
  var debugEl = document.getElementById('debug');
  function log(msg) {
    debugEl.innerHTML += '<br/>' + msg;
  }
  var video = document.getElementById('video');
  var src = "%s";
  log("Target URL: " + src.substring(0,50) + "...");
  
  if(Hls.isSupported()){
    log("Hls.js is supported");
    var hls = new Hls({
      maxBufferLength: 30,
      maxMaxBufferLength: 60,
    });
    hls.loadSource(src);
    hls.attachMedia(video);
    hls.on(Hls.Events.MANIFEST_PARSED, function(){
      log("Manifest parsed. Trying to play...");
      video.play().then(()=>log("Playback started")).catch(function(e){ log("Play error: " + e.message); });
    });
    hls.on(Hls.Events.ERROR, function(event, data){
      log("HLS Error: " + data.type + " - " + data.details);
      if(data.fatal){
        switch(data.type){
          case Hls.ErrorTypes.NETWORK_ERROR:
            log("Fatal network error, recovering...");
            hls.startLoad();
            break;
          case Hls.ErrorTypes.MEDIA_ERROR:
            log("Fatal media error, recovering...");
            hls.recoverMediaError();
            break;
          default:
            hls.destroy();
            break;
        }
      }
    });
  } else if(video.canPlayType('application/vnd.apple.mpegurl')){
    log("Native HLS supported");
    video.src = src;
    video.addEventListener('loadedmetadata', function(){
      video.play().catch(function(e){ log("Play error: " + e.message); });
    });
  } else {
    log("HLS is NOT supported in this browser!");
  }
})();
</script>
</body>
</html>`, m3u8URL)
}

// ProxyVideoEmbed fetches a video embed page, extracts the raw m3u8 stream URL,
// and serves a completely clean video player with zero ads.
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

	// Try to extract the m3u8 stream URL
	m3u8URL, err := extractM3U8FromEmbed(rawURL)
	if err != nil {
		// Fallback: proxy the original page as-is (for non-packed embed pages)
		fallbackProxy(c, rawURL, parsedURL)
		return
	}

	// Serve a clean player page
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Frame-Options", "ALLOWALL")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(cleanPlayerHTML(m3u8URL)))
}

// fallbackProxy is the old behavior: fetch and forward the embed page for
// video providers that don't use packed scripts (e.g. filedon.co)
func fallbackProxy(c *gin.Context, rawURL string, parsedURL *url.URL) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch video page"})
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	if !strings.Contains(contentType, "text/html") {
		c.Header("Access-Control-Allow-Origin", "*")
		c.DataFromReader(resp.StatusCode, resp.ContentLength, contentType, resp.Body, nil)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}

	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Frame-Options", "ALLOWALL")
	c.Data(http.StatusOK, "text/html; charset=utf-8", bodyBytes)
}
