package update

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{left: "v0.10.0", right: "v0.9.0", want: compareNewer},
		{left: "v1.0.0", right: "v1.0.0", want: compareEqual},
		{left: "v1.0.0-beta.1", right: "v1.0.0", want: -1},
		{left: "v1.0.0", right: "v1.0.0-beta.1", want: compareNewer},
	}

	for _, test := range tests {
		t.Run(test.left+"_"+test.right, func(t *testing.T) {
			got, err := compareVersions(test.left, test.right)
			if err != nil {
				t.Fatalf("compareVersions() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("compareVersions() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestCheckUsesGitHubLatestReleaseRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/latest" {
			t.Fatalf("path = %s, want /releases/latest", r.URL.Path)
		}
		http.Redirect(w, r, "/owner/repo/releases/tag/v0.2.0", http.StatusFound)
	}))
	defer server.Close()
	t.Setenv(envReleaseBase, server.URL)

	info, err := Check("v0.1.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !info.Available {
		t.Fatal("expected update to be available")
	}
	if info.LatestVersion != "v0.2.0" {
		t.Fatalf("LatestVersion = %q, want v0.2.0", info.LatestVersion)
	}
	if !strings.Contains(info.DownloadURL, "/releases/download/v0.2.0/futu_v0.2.0_") {
		t.Fatalf("DownloadURL = %q", info.DownloadURL)
	}
	if info.ChecksumURL != server.URL+"/releases/download/v0.2.0/futu_checksums.txt" {
		t.Fatalf("ChecksumURL = %q", info.ChecksumURL)
	}
	if info.ReleaseNotes != server.URL+"/releases/tag/v0.2.0" {
		t.Fatalf("ReleaseNotes = %q", info.ReleaseNotes)
	}
}

func TestDefaultConfigUsesDefaultReleaseBase(t *testing.T) {
	t.Setenv(envReleaseBase, "")

	cfg, err := defaultConfig()
	if err != nil {
		t.Fatalf("defaultConfig() error = %v", err)
	}
	if cfg.ReleaseBase != defaultReleaseBase {
		t.Fatalf("ReleaseBase = %q, want %q", cfg.ReleaseBase, defaultReleaseBase)
	}
}

func TestRefreshCacheSkipsDevVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(envReleaseBase, "")

	RefreshCacheIfStale("dev")
	if _, err := os.Stat(filepath.Join(home, ".futu", latestCacheFile)); !os.IsNotExist(err) {
		t.Fatalf("latest cache was created: %v", err)
	}
}

func TestFindChecksum(t *testing.T) {
	content := "abc123  futu_v0.1.0_darwin_arm64.tar.gz\n"
	got := findChecksum(content, "futu_v0.1.0_darwin_arm64.tar.gz")
	if got != "abc123" {
		t.Fatalf("checksum = %q, want abc123", got)
	}
}

func TestTagFromLocation(t *testing.T) {
	got, err := tagFromLocation("https://github.com/owner/repo/releases/tag/v1.2.3?expanded=true")
	if err != nil {
		t.Fatalf("tagFromLocation() error = %v", err)
	}
	if got != "v1.2.3" {
		t.Fatalf("tag = %q, want v1.2.3", got)
	}
}

func TestTagFromLocationRejectsReleasesPage(t *testing.T) {
	if _, err := tagFromLocation("https://github.com/owner/repo/releases"); err == nil {
		t.Fatal("expected latest stable release error")
	}
}
