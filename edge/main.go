package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CacheItem Simple in-memory cache
type CacheItem struct {
	Data      []byte
	Header    http.Header
	ExpiresAt time.Time
}

var cache = make(map[string]*CacheItem)
var cacheMutex sync.RWMutex

// Domain → Origin mapping
var origins = map[string]string{
	"example.com": "http://localhost:8081", // origin server 1
	"test.com":    "http://localhost:8082", // origin server 2
}

// Cache TTL
const cacheTTL = 30 * time.Second

func main() {
	r := gin.Default()

	r.Any("/*path", handleRequest)

	fmt.Println("Edge service running on :8080")
	err := r.Run(":8080")
	if err != nil {
		fmt.Println(err)
	}
}

func handleRequest(c *gin.Context) {
	host := c.Request.Host
	origin, ok := origins[host]
	if !ok {
		c.String(http.StatusBadGateway, "Unknown host: %s", host)
		return
	}

	cacheKey := host + c.Request.URL.Path

	// 1. Check cache
	cacheMutex.RLock()
	item, found := cache[cacheKey]
	cacheMutex.RUnlock()

	if found && time.Now().Before(item.ExpiresAt) {
		// Cache hit
		for k, vals := range item.Header {
			for _, v := range vals {
				c.Writer.Header().Add(k, v)
			}
		}
		c.Data(http.StatusOK, item.Header.Get("Content-Type"), item.Data)
		return
	}

	// 2. Fetch from origin
	targetURL := origin + c.Request.URL.Path
	resp, err := http.Get(targetURL)
	if err != nil {
		c.String(http.StatusBadGateway, "Error fetching from origin: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 3. Save to cache
	cacheMutex.Lock()
	cache[cacheKey] = &CacheItem{
		Data:      body,
		Header:    resp.Header.Clone(),
		ExpiresAt: time.Now().Add(cacheTTL),
	}
	cacheMutex.Unlock()

	// 4. Return response
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
