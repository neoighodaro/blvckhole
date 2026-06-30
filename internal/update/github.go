package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL  = "https://api.github.com"
	defaultRepo     = "neoighodaro/blvckhole"
	apiTimeout      = 5 * time.Second  // budget for small JSON API calls
	downloadTimeout = 60 * time.Second // budget for multi-MB tarball downloads
)

// Asset is a single downloadable file attached to a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Release is a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// AssetURL returns the download URL for the asset with the given name.
func (r Release) AssetURL(name string) (string, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.URL, true
		}
	}
	return "", false
}

// Client talks to the GitHub releases API.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	Repo       string
}

// NewClient returns a Client with sensible defaults. The underlying http.Client
// uses downloadTimeout (60 s) to accommodate multi-MB tarball downloads.
// LatestRelease further constrains its context to apiTimeout (5 s) so the
// small JSON call stays fast regardless of the client-level timeout.
func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: downloadTimeout},
		BaseURL:    defaultBaseURL,
		Repo:       defaultRepo,
	}
}

// LatestRelease fetches the latest non-prerelease, non-draft release.
// It applies a short apiTimeout deadline on top of the caller's context so the
// JSON round-trip stays fast even though the client allows a longer download.
func (c *Client) LatestRelease(ctx context.Context) (Release, error) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.BaseURL, c.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("github: unexpected status %s", resp.Status)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return Release{}, err
	}
	return rel, nil
}

// Download fetches the body at url. Non-200 responses are errors.
func (c *Client) Download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// AssetName builds the release tarball name for a version/os/arch triple.
func AssetName(version, goos, goarch string) string {
	return fmt.Sprintf("blvckhole-%s-%s-%s.tar.gz", version, goos, goarch)
}
