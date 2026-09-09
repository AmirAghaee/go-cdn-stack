package filesystemstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

func TestSaveAndLoadSnapshot(t *testing.T) {
	item, err := cdn.New("id", "CDN.Example", "http://origin.example", true, 60)
	if err != nil {
		t.Fatal(err)
	}
	store := New(filepath.Join(t.TempDir(), "snapshots", "cdns.json"))
	if err := store.Save(context.Background(), []cdn.CDN{item}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	items, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(items) != 1 || items[0].Domain() != "cdn.example" || items[0].Origin() != item.Origin() {
		t.Fatalf("Load() = %#v", items)
	}
}
