package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type CacheItem struct {
	FilePath  string
	Header    http.Header
	ExpiresAt time.Time
}

var cache = make(map[string]*CacheItem)
var cacheMutex sync.RWMutex

var origins = map[string]string{
	"example.com": "http://localhost:8081", // origin server 1
	"test.com":    "http://localhost:8082", // origin server 2
}

const (
	cacheTTL    = 180 * time.Second
	cacheDir    = "./cache"
	metadataExt = ".json"
)

func main() {
	// Ensure cache dir exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		panic(err)
	}

	// Load previously cached files
	loadCacheFromDisk()

	r := gin.Default()
	r.Any("/*path", handleRequest)

	fmt.Println("Edge service running on :8080")
	if err := r.Run(":8080"); err != nil {
		panic(err)
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
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%x.cache", cacheKey))

	// 1. Check in-memory cache
	cacheMutex.RLock()
	item, found := cache[cacheKey]
	cacheMutex.RUnlock()

	if found && time.Now().Before(item.ExpiresAt) {
		serveFromFile(c, item)
		return
	}

	// 2. Fetch from origin
	targetURL := origin + c.Request.URL.Path
	resp, err := http.Get(targetURL)
	if err != nil {
		c.String(http.StatusBadGateway, "Error fetching from origin: %v", err)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
		return
	}
	defer resp.Body.Close()

	// Write body to file
	if err := os.WriteFile(cacheFile, body, 0644); err != nil {
		c.String(http.StatusInternalServerError, "Error writing cache file: %v", err)
		return
	}

	// Save metadata
	meta := &CacheItem{
		FilePath:  cacheFile,
		Header:    resp.Header.Clone(),
		ExpiresAt: time.Now().Add(cacheTTL),
	}
	metaFile := cacheFile + metadataExt
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaFile, metaJSON, 0644)

	cacheMutex.Lock()
	cache[cacheKey] = meta
	cacheMutex.Unlock()

	// Return to client
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

func serveFromFile(c *gin.Context, item *CacheItem) {
	body, err := os.ReadFile(item.FilePath)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading cache file: %v", err)
		return
	}

	for k, vals := range item.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Data(http.StatusOK, item.Header.Get("Content-Type"), body)
}

func loadCacheFromDisk() {
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		fmt.Println("Error reading cache dir:", err)
		return
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		name := f.Name()
		if !strings.HasSuffix(name, ".cache.json") {
			continue
		}

		metaFile := filepath.Join(cacheDir, name)
		data, err := os.ReadFile(metaFile)
		if err != nil {
			fmt.Println("Error reading cache metadata:", err)
			continue
		}

		var item CacheItem
		if err := json.Unmarshal(data, &item); err != nil {
			fmt.Println("Error parsing cache metadata:", err)
			continue
		}

		// Only load if not expired
		if time.Now().Before(item.ExpiresAt) {
			// Extract original cacheKey from file name
			// e.g. "6578616d706c652e636f6d2f696d6167652e6a7067.cache.json" → decode hex
			hexName := strings.TrimSuffix(name, ".cache.json")
			keyBytes, err := hex.DecodeString(hexName)
			if err != nil {
				fmt.Println("Error decoding hex filename:", err)
				continue
			}
			cacheKey := string(keyBytes)

			cacheMutex.Lock()
			cache[cacheKey] = &item
			cacheMutex.Unlock()
		}
	}
	fmt.Printf("Loaded %d cache items from disk\n", len(cache))
}
