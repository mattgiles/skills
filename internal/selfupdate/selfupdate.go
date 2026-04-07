package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	defaultOwner  = "mattgiles"
	defaultRepo   = "skills"
	defaultBinary = "skills"
)

type Options struct {
	CurrentVersion string
	TargetVersion  string
	TargetPath     string
	HTTPClient     *http.Client
	APIURL         string
	ReleaseBaseURL string
}

type Result struct {
	PreviousVersion string
	Version         string
	TargetPath      string
	Updated         bool
}

type latestReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func Run(options Options) (Result, error) {
	targetPath := strings.TrimSpace(options.TargetPath)
	if targetPath == "" {
		return Result{}, errors.New("target path is required")
	}

	osName, arch, err := detectPlatform()
	if err != nil {
		return Result{}, err
	}

	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	version := strings.TrimSpace(options.TargetVersion)
	if version == "" {
		version, err = resolveLatestVersion(client, apiURL(options.APIURL))
		if err != nil {
			return Result{}, err
		}
	}

	result := Result{
		PreviousVersion: strings.TrimSpace(options.CurrentVersion),
		Version:         version,
		TargetPath:      targetPath,
	}
	if result.PreviousVersion != "" && result.PreviousVersion == version {
		return result, nil
	}

	releaseBase := releaseBaseURL(options.ReleaseBaseURL, version)
	assetVersion := normalizeAssetVersion(version)
	assetName := fmt.Sprintf("%s_%s_%s_%s.tar.gz", defaultBinary, assetVersion, osName, arch)
	checksumsName := fmt.Sprintf("%s_checksums.txt", defaultBinary)

	tmpdir, err := os.MkdirTemp("", "skills-self-update-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(tmpdir)

	archivePath := filepath.Join(tmpdir, assetName)
	checksumsPath := filepath.Join(tmpdir, checksumsName)
	binaryPath := filepath.Join(tmpdir, defaultBinary)

	if err := downloadToFile(client, releaseBase+"/"+assetName, archivePath); err != nil {
		return Result{}, err
	}
	if err := downloadToFile(client, releaseBase+"/"+checksumsName, checksumsPath); err != nil {
		return Result{}, err
	}
	if err := verifyChecksum(archivePath, checksumsPath); err != nil {
		return Result{}, err
	}
	if err := extractBinary(archivePath, binaryPath); err != nil {
		return Result{}, err
	}
	if err := replaceBinary(binaryPath, targetPath); err != nil {
		return Result{}, err
	}

	result.Updated = true
	return result, nil
}

func detectPlatform() (string, string, error) {
	if runtime.GOOS != "darwin" {
		return "", "", fmt.Errorf("unsupported operating system: %s (v1 supports macOS only)", runtime.GOOS)
	}

	switch runtime.GOARCH {
	case "arm64", "amd64":
		return runtime.GOOS, runtime.GOARCH, nil
	default:
		return "", "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
}

func resolveLatestVersion(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve latest release: unexpected status %s", resp.Status)
	}

	var payload latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return "", errors.New("resolve latest release: empty tag_name")
	}
	return payload.TagName, nil
}

func apiURL(override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimRight(override, "/")
	}
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", defaultOwner, defaultRepo)
}

func releaseBaseURL(override string, version string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimRight(override, "/") + "/" + version
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", defaultOwner, defaultRepo, version)
}

func normalizeAssetVersion(version string) string {
	if strings.HasPrefix(version, "v") {
		return strings.TrimPrefix(version, "v")
	}
	return version
}

func downloadToFile(client *http.Client, url string, path string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return err
	}
	return nil
}

func verifyChecksum(archivePath string, checksumsPath string) error {
	archiveName := filepath.Base(archivePath)
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return err
	}

	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == archiveName {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("missing checksum for %s", archiveName)
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s", archiveName)
	}
	return nil
}

func extractBinary(archivePath string, outputPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(header.Name) != defaultBinary {
			continue
		}

		out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("archive did not contain %s", defaultBinary)
}

func replaceBinary(newBinaryPath string, targetPath string) error {
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		return err
	}

	tmpTarget := targetPath + ".tmp"
	if err := copyFile(newBinaryPath, tmpTarget, targetInfo.Mode().Perm()); err != nil {
		return err
	}
	if err := os.Rename(tmpTarget, targetPath); err != nil {
		_ = os.Remove(tmpTarget)
		return err
	}
	return nil
}

func copyFile(src string, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return nil
}
