package filesystemstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
)

type Purger struct {
	directory string
}

func NewPurger(directory string) (*Purger, error) {
	absoluteDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve cache directory: %w", err)
	}
	return &Purger{directory: absoluteDirectory}, nil
}

func (p *Purger) PurgeDomain(ctx context.Context, domain string) (cache.PurgeResult, error) {
	result := cache.PurgeResult{Domain: domain}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	files, err := os.ReadDir(p.directory)
	if err != nil {
		return result, fmt.Errorf("read cache directory: %w", err)
	}

	var purgeErrors []error
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			purgeErrors = append(purgeErrors, err)
			break
		}
		if file.IsDir() || !isCacheMetadata(file.Name()) {
			continue
		}

		metadataFile := filepath.Join(p.directory, file.Name())
		data, err := os.ReadFile(metadataFile)
		if err != nil {
			continue
		}
		var item metadata
		if err := json.Unmarshal(data, &item); err != nil ||
			filepath.Clean(metadataFile) != metadataPath(p.directory, item.Key) ||
			!validMetadata(p.directory, item.Key, item) {
			continue
		}
		itemDomain, ok := cache.DomainFromKey(item.Key)
		if !ok || itemDomain != domain {
			continue
		}

		var bodySize int64
		if info, statErr := os.Lstat(item.FilePath); statErr == nil && info.Mode().IsRegular() {
			bodySize = info.Size()
		}
		if err := os.Remove(item.FilePath); err != nil && !os.IsNotExist(err) {
			purgeErrors = append(purgeErrors, fmt.Errorf("remove cache body %q: %w", item.FilePath, err))
		} else if err == nil {
			result.BytesFreed += bodySize
		}
		if err := os.Remove(metadataFile); err != nil && !os.IsNotExist(err) {
			purgeErrors = append(purgeErrors, fmt.Errorf("remove cache metadata %q: %w", metadataFile, err))
		} else {
			result.EntriesPurged++
		}
	}
	return result, errors.Join(purgeErrors...)
}
