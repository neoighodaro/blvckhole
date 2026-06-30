package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssetName(t *testing.T) {
	got := AssetName("v0.0.5", "darwin", "arm64")
	want := "blvckhole-v0.0.5-darwin-arm64.tar.gz"
	if got != want {
		t.Errorf("AssetName = %q, want %q", got, want)
	}
}

func TestLatestReleaseAndAssetURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/neoighodaro/blvckhole/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{
			"tag_name": "v0.0.5",
			"assets": [
				{"name": "blvckhole-v0.0.5-darwin-arm64.tar.gz", "browser_download_url": "https://example/dl/mac"},
				{"name": "checksums.txt", "browser_download_url": "https://example/dl/sums"}
			]
		}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	rel, err := c.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.TagName != "v0.0.5" {
		t.Errorf("TagName = %q, want v0.0.5", rel.TagName)
	}
	url, ok := rel.AssetURL("blvckhole-v0.0.5-darwin-arm64.tar.gz")
	if !ok || url != "https://example/dl/mac" {
		t.Errorf("AssetURL = %q,%v want https://example/dl/mac,true", url, ok)
	}
	if _, ok := rel.AssetURL("nonexistent.tar.gz"); ok {
		t.Error("AssetURL found a nonexistent asset")
	}
}

func TestDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.Write([]byte("payload"))
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient()
	data, err := c.Download(context.Background(), srv.URL+"/ok")
	if err != nil || string(data) != "payload" {
		t.Errorf("Download ok = %q,%v", data, err)
	}
	if _, err := c.Download(context.Background(), srv.URL+"/missing"); err == nil {
		t.Error("Download of 404 should error")
	}
}
