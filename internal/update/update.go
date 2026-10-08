package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/FutunnOpen/futu-cli/internal/config"
)

const (
	productName = "futu"
	binaryName  = "futu"
	npmPackage  = "@futunn/futu-cli"

	envReleaseBase = "FUTU_CLI_RELEASE_BASE"
	envLibc        = "FUTU_CLI_LIBC"

	latestCacheFile     = ".cli-latest-version"
	lastRunVersionFile  = ".cli-last-run-version"
	checksumFileName    = "futu_checksums.txt"
	defaultReleaseBase  = "https://github.com/FutunnOpen/futu-cli"
	checkInterval       = 24 * time.Hour
	checkTimeout        = 2 * time.Second
	downloadTimeout     = 300 * time.Second
	dirPermissions      = 0o700
	filePermissions     = 0o600
	executablePerms     = 0o755
	versionParts        = 3
	compareNewer        = 1
	compareEqual        = 0
	defaultLinuxLibc    = "musl"
	linuxLibcGlibc      = "glibc"
	archiveExtTarGzip   = "tar.gz"
	archiveExtZip       = "zip"
	windowsExecutable   = ".exe"
	npmInstallHint      = "This CLI was installed by npm. Run `npm update -g @futunn/futu-cli` to upgrade."
	homebrewInstallHint = "This CLI appears to be managed by Homebrew. Run `brew upgrade futu` to upgrade."
)

// UpdateInfo holds the result of a version check against the release source.
type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	DownloadURL    string `json:"download_url"`
	ChecksumURL    string `json:"checksum_url"`
	AssetName      string `json:"asset_name"`
	ReleaseNotes   string `json:"release_notes"`
	Available      bool   `json:"available"`
}

type releaseConfig struct {
	ReleaseBase string
	LatestURL   string
}

type assetInfo struct {
	Name string
	Ext  string
}

// Check queries the GitHub latest release redirect to determine whether a newer
// version exists.
func Check(currentVersion string) (*UpdateInfo, error) {
	latest, cfg, err := fetchLatestVersion(context.Background(), downloadTimeout)
	if err != nil {
		return nil, err
	}
	return buildUpdateInfo(currentVersion, latest, cfg)
}

// Apply downloads and installs the update described by info.
func Apply(info *UpdateInfo) error {
	if info == nil || !info.Available {
		return fmt.Errorf("no update available")
	}
	return applyVersion(info.LatestVersion)
}

// ReleaseNotes returns the current GitHub release page URL.
func ReleaseNotes() (string, error) {
	latest, cfg, err := fetchLatestVersion(context.Background(), downloadTimeout)
	if err != nil {
		return "", err
	}
	return cfg.releaseNotesURL(latest), nil
}

func releaseNotesURLForVersion(version string) string {
	cfg, err := defaultConfig()
	if err != nil {
		return ""
	}
	return cfg.releaseNotesURL(version)
}

// RefreshCacheIfStale refreshes the latest-version cache in the background.
func RefreshCacheIfStale(currentVersion string) {
	if skipVersionCheck(currentVersion) || cacheFresh(latestCachePath(), time.Now()) {
		return
	}
	if _, err := defaultConfig(); err != nil {
		return
	}
	_ = writeVersionFile(latestCachePath(), normalizeVersion(currentVersion))
	go refreshLatestCache()
}

// NotifyIfAvailable prints a cached update notice to stderr when a newer version
// is available. It performs only local disk I/O and suppresses all errors.
func NotifyIfAvailable(currentVersion string) {
	if skipVersionCheck(currentVersion) || !stderrIsTerminal() {
		return
	}
	latest, err := readVersionFile(latestCachePath())
	if err != nil || len(latest) == 0 {
		return
	}
	if newer, err := isVersionNewer(latest, currentVersion); err == nil && newer {
		fmt.Fprintf(os.Stderr, "New version %s is available, run `futu update` to update.\n", latest)
		if url := releaseNotesURLForVersion(latest); len(url) > 0 {
			fmt.Fprintf(os.Stderr, "Release notes: %s\n", url)
		}
	}
}

// NotifyReleaseNotesIfVersionChanged prints release notes once after the binary
// version changes, including upgrades done by package managers.
func NotifyReleaseNotesIfVersionChanged(currentVersion string) {
	if skipVersionCheck(currentVersion) || !stderrIsTerminal() {
		return
	}
	path := filepath.Join(config.ConfigDir(), lastRunVersionFile)
	previous, _ := readVersionFile(path)
	_ = writeVersionFile(path, normalizeVersion(currentVersion))
	if len(previous) == 0 || previous == normalizeVersion(currentVersion) {
		return
	}
	if url := releaseNotesURLForVersion(currentVersion); len(url) > 0 {
		fmt.Fprintf(os.Stderr, "Release notes: %s\n", url)
	}
}

func applyVersion(version string) error {
	cfg, err := defaultConfig()
	if err != nil {
		return err
	}
	asset, err := currentAsset(version)
	if err != nil {
		return err
	}
	exe, err := currentExecutable()
	if err != nil {
		return err
	}
	if hint := managedInstallHint(exe); len(hint) > 0 {
		return errors.New(hint)
	}
	tmpDir, err := os.MkdirTemp("", productName+"-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, asset.Name)
	if err := downloadFile(cfg.assetURL(version, asset.Name), archivePath, downloadTimeout); err != nil {
		return err
	}
	if err := verifyChecksum(cfg.checksumURL(version), archivePath, asset.Name); err != nil {
		return err
	}
	binPath, err := extractBinary(archivePath, asset.Ext, tmpDir)
	if err != nil {
		return err
	}
	if err := verifyBinaryVersion(binPath, version); err != nil {
		return err
	}
	if err := replaceExecutable(binPath, exe); err != nil {
		return err
	}
	_ = os.Remove(latestCachePath())
	return nil
}

func buildUpdateInfo(currentVersion, latest string, cfg releaseConfig) (*UpdateInfo, error) {
	asset, err := currentAsset(latest)
	if err != nil {
		return nil, err
	}
	available, err := isVersionNewer(latest, currentVersion)
	if err != nil {
		return nil, err
	}
	return &UpdateInfo{
		CurrentVersion: normalizeVersion(currentVersion),
		LatestVersion:  normalizeVersion(latest),
		DownloadURL:    cfg.assetURL(latest, asset.Name),
		ChecksumURL:    cfg.checksumURL(latest),
		AssetName:      asset.Name,
		ReleaseNotes:   cfg.releaseNotesURL(latest),
		Available:      available,
	}, nil
}

func fetchLatestVersion(ctx context.Context, timeout time.Duration) (string, releaseConfig, error) {
	cfg, err := defaultConfig()
	if err != nil {
		return "", cfg, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	version, err := fetchLatestTag(ctx, cfg.LatestURL)
	if err != nil {
		return "", cfg, fmt.Errorf("fetch latest version: %w", err)
	}
	if _, err := parseSemVersion(version); err != nil {
		return "", cfg, fmt.Errorf("invalid latest version %q: %w", version, err)
	}
	return version, cfg, nil
}

func refreshLatestCache() {
	latest, _, err := fetchLatestVersion(context.Background(), checkTimeout)
	if err != nil {
		return
	}
	_ = writeVersionFile(latestCachePath(), latest)
}

func defaultConfig() (releaseConfig, error) {
	base := strings.TrimRight(os.Getenv(envReleaseBase), "/")
	if len(base) == 0 {
		base = strings.TrimRight(defaultReleaseBase, "/")
	}
	if len(base) == 0 {
		return releaseConfig{}, fmt.Errorf("GitHub release base is not configured; set %s or defaultReleaseBase", envReleaseBase)
	}
	return releaseConfig{
		ReleaseBase: base,
		LatestURL:   base + "/releases/latest",
	}, nil
}

func (c releaseConfig) assetURL(version, asset string) string {
	return c.ReleaseBase + "/releases/download/" + normalizeVersion(version) + "/" + asset
}

func (c releaseConfig) checksumURL(version string) string {
	return c.assetURL(version, checksumFileName)
}

func (c releaseConfig) releaseNotesURL(version string) string {
	return c.ReleaseBase + "/releases/tag/" + normalizeVersion(version)
}

func currentAsset(version string) (assetInfo, error) {
	osName, arch := runtime.GOOS, runtime.GOARCH
	platform, ext, err := platformSuffix(osName, arch)
	if err != nil {
		return assetInfo{}, err
	}
	return assetInfo{
		Name: fmt.Sprintf("%s_%s_%s.%s", binaryName, normalizeVersion(version), platform, ext),
		Ext:  ext,
	}, nil
}

func platformSuffix(osName, arch string) (string, string, error) {
	switch osName {
	case "darwin":
		return osName + "_" + arch, archiveExtTarGzip, requireArch(arch)
	case "linux":
		return linuxPlatformSuffix(arch), archiveExtTarGzip, requireArch(arch)
	case "windows":
		if arch != "amd64" {
			return "", "", unsupportedPlatform(osName, arch)
		}
		return osName + "_" + arch, archiveExtZip, nil
	default:
		return "", "", unsupportedPlatform(osName, arch)
	}
}

func linuxPlatformSuffix(arch string) string {
	if strings.EqualFold(os.Getenv(envLibc), linuxLibcGlibc) {
		return "linux_" + arch
	}
	return "linux_" + defaultLinuxLibc + "_" + arch
}

func requireArch(arch string) error {
	if arch != "amd64" && arch != "arm64" {
		return unsupportedPlatform(runtime.GOOS, arch)
	}
	return nil
}

func unsupportedPlatform(osName, arch string) error {
	return fmt.Errorf("unsupported platform: %s/%s", osName, arch)
}

func getBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func fetchLatestTag(ctx context.Context, latestURL string) (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
		return tagFromLocation(resp.Header.Get("Location"))
	}
	if resp.StatusCode == http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return normalizeVersion(string(data)), nil
	}
	return "", fmt.Errorf("unexpected status: %s", resp.Status)
}

func tagFromLocation(location string) (string, error) {
	if len(location) == 0 {
		return "", fmt.Errorf("missing latest release redirect location")
	}
	clean := strings.TrimRight(strings.Split(location, "?")[0], "/")
	index := strings.LastIndex(clean, "/")
	if index < 0 || index == len(clean)-1 {
		return "", fmt.Errorf("invalid latest release redirect location: %s", location)
	}
	return normalizeVersion(clean[index+1:]), nil
}

func downloadFile(url, path string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	data, err := getBytes(ctx, url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	if err := os.WriteFile(path, data, filePermissions); err != nil {
		return fmt.Errorf("write download: %w", err)
	}
	return nil
}

func verifyChecksum(checksumURL, archivePath, assetName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	data, err := getBytes(ctx, checksumURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	expected := findChecksum(string(data), assetName)
	if len(expected) == 0 {
		return fmt.Errorf("checksum for %s not found", assetName)
	}
	actual, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum mismatch for %s", assetName)
	}
	return nil
}

func findChecksum(content, assetName string) string {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == assetName {
			return fields[0]
		}
	}
	return ""
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash archive: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func extractBinary(archivePath, ext, tmpDir string) (string, error) {
	switch ext {
	case archiveExtTarGzip:
		return extractTarGzipBinary(archivePath, tmpDir)
	case archiveExtZip:
		return extractZipBinary(archivePath, tmpDir)
	default:
		return "", fmt.Errorf("unsupported archive extension: %s", ext)
	}
}

func extractTarGzipBinary(archivePath, tmpDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("open gzip: %w", err)
	}
	defer gzipReader.Close()
	return extractTarEntry(tar.NewReader(gzipReader), tmpDir)
}

func extractTarEntry(reader *tar.Reader, tmpDir string) (string, error) {
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read tar: %w", err)
		}
		if filepath.Base(header.Name) != binaryName {
			continue
		}
		return writeExtractedBinary(reader, tmpDir, binaryName)
	}
	return "", fmt.Errorf("%s not found in archive", binaryName)
}

func extractZipBinary(archivePath, tmpDir string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()
	target := binaryName + windowsExecutable
	for _, file := range reader.File {
		if filepath.Base(file.Name) != target {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("open zip entry: %w", err)
		}
		defer rc.Close()
		return writeExtractedBinary(rc, tmpDir, target)
	}
	return "", fmt.Errorf("%s not found in archive", target)
}

func writeExtractedBinary(reader io.Reader, tmpDir, name string) (string, error) {
	path := filepath.Join(tmpDir, name)
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, executablePerms)
	if err != nil {
		return "", fmt.Errorf("create binary: %w", err)
	}
	if _, err := io.Copy(out, reader); err != nil {
		_ = out.Close()
		return "", fmt.Errorf("write binary: %w", err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("close binary: %w", err)
	}
	if err := os.Chmod(path, executablePerms); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}
	return path, nil
}

func verifyBinaryVersion(binPath, version string) error {
	out, err := exec.Command(binPath, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify binary version: %w", err)
	}
	if !strings.Contains(string(out), normalizeVersion(version)) {
		return fmt.Errorf("binary version output %q does not contain %s", strings.TrimSpace(string(out)), version)
	}
	return nil
}

func replaceExecutable(newBinary, current string) error {
	if runtime.GOOS == "windows" {
		return replaceWindowsExecutable(newBinary, current)
	}
	return replaceUnixExecutable(newBinary, current)
}

func replaceUnixExecutable(newBinary, current string) error {
	tmpTarget := current + ".new"
	if err := copyFile(newBinary, tmpTarget, executablePerms); err != nil {
		return err
	}
	if err := os.Rename(tmpTarget, current); err != nil {
		_ = os.Remove(tmpTarget)
		return fmt.Errorf("replace executable: %w", err)
	}
	return nil
}

func replaceWindowsExecutable(newBinary, current string) error {
	oldPath := current + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(current, oldPath); err != nil {
		return fmt.Errorf("backup current executable: %w", err)
	}
	if err := copyFile(newBinary, current, executablePerms); err != nil {
		_ = os.Rename(oldPath, current)
		return err
	}
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	input, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source binary: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("create target binary: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return fmt.Errorf("copy target binary: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close target binary: %w", err)
	}
	return os.Chmod(dst, perm)
}

func currentExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve current executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err == nil {
		return resolved, nil
	}
	return exe, nil
}

func managedInstallHint(exe string) string {
	lower := strings.ToLower(filepath.ToSlash(exe))
	switch {
	case strings.Contains(lower, "node_modules"):
		return npmInstallHint
	case strings.Contains(lower, "homebrew") || strings.Contains(lower, "/cellar/"):
		return homebrewInstallHint
	default:
		return ""
	}
}

func cacheFresh(path string, now time.Time) bool {
	info, err := os.Stat(path)
	return err == nil && now.Sub(info.ModTime()) < checkInterval
}

func latestCachePath() string {
	return filepath.Join(config.ConfigDir(), latestCacheFile)
}

func readVersionFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return normalizeVersion(string(data)), nil
}

func writeVersionFile(path, version string) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPermissions); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(normalizeVersion(version)+"\n"), filePermissions)
}

func isVersionNewer(candidate, current string) (bool, error) {
	if strings.TrimSpace(current) == "dev" {
		return true, nil
	}
	cmp, err := compareVersions(candidate, current)
	if err != nil {
		return false, err
	}
	return cmp == compareNewer, nil
}

func compareVersions(left, right string) (int, error) {
	lv, err := parseSemVersion(left)
	if err != nil {
		return 0, err
	}
	rv, err := parseSemVersion(right)
	if err != nil {
		return 0, err
	}
	for i := 0; i < versionParts; i++ {
		if lv.parts[i] > rv.parts[i] {
			return compareNewer, nil
		}
		if lv.parts[i] < rv.parts[i] {
			return -1, nil
		}
	}
	return comparePrerelease(lv.prerelease, rv.prerelease), nil
}

type semVersion struct {
	parts      [versionParts]int
	prerelease string
}

func parseSemVersion(raw string) (semVersion, error) {
	version := strings.TrimPrefix(normalizeVersion(raw), "v")
	base, prerelease, _ := strings.Cut(version, "-")
	fields := strings.Split(base, ".")
	if len(fields) != versionParts {
		return semVersion{}, fmt.Errorf("expected major.minor.patch")
	}
	var parsed semVersion
	parsed.prerelease = prerelease
	for i, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil || value < 0 {
			return semVersion{}, fmt.Errorf("invalid numeric component %q", field)
		}
		parsed.parts[i] = value
	}
	return parsed, nil
}

func comparePrerelease(left, right string) int {
	switch {
	case left == right:
		return compareEqual
	case len(left) == 0:
		return compareNewer
	case len(right) == 0:
		return -1
	default:
		if left > right {
			return compareNewer
		}
		return -1
	}
}

func normalizeVersion(version string) string {
	return strings.TrimSpace(version)
}

func skipVersionCheck(currentVersion string) bool {
	return len(normalizeVersion(currentVersion)) == 0 || normalizeVersion(currentVersion) == "dev"
}

func stderrIsTerminal() bool {
	info, err := os.Stderr.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}
