package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunDownloadsAndReplacesBinary(t *testing.T) {
	archive := buildArchive(t, "new-binary")
	sum := sha256.Sum256(archive)
	checksums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), "skills_1.2.3_darwin_"+runtimeArch()+".tar.gz")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/latest":
			_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
		case "/releases/v1.2.3/skills_1.2.3_darwin_" + runtimeArch() + ".tar.gz":
			_, _ = w.Write(archive)
		case "/releases/v1.2.3/skills_checksums.txt":
			_, _ = w.Write([]byte(checksums))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "skills")
	if err := os.WriteFile(targetPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := Run(Options{
		CurrentVersion: "v1.2.2",
		TargetPath:     targetPath,
		APIURL:         server.URL + "/api/latest",
		ReleaseBaseURL: server.URL + "/releases",
		HTTPClient:     server.Client(),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Updated || result.Version != "v1.2.3" {
		t.Fatalf("unexpected result: %+v", result)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "new-binary" {
		t.Fatalf("binary contents = %q", string(data))
	}
}

func TestRunNoopsWhenAlreadyCurrent(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "skills")
	if err := os.WriteFile(targetPath, []byte("current"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := Run(Options{
		CurrentVersion: "v1.2.3",
		TargetVersion:  "v1.2.3",
		TargetPath:     targetPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Updated {
		t.Fatalf("expected no update, got %+v", result)
	}
}

func TestRunFailsOnChecksumMismatch(t *testing.T) {
	archive := buildArchive(t, "new-binary")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/v1.2.3/skills_1.2.3_darwin_" + runtimeArch() + ".tar.gz":
			_, _ = w.Write(archive)
		case "/releases/v1.2.3/skills_checksums.txt":
			_, _ = w.Write([]byte("deadbeef  skills_1.2.3_darwin_" + runtimeArch() + ".tar.gz\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	targetPath := filepath.Join(t.TempDir(), "skills")
	if err := os.WriteFile(targetPath, []byte("current"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Run(Options{
		CurrentVersion: "v1.2.2",
		TargetVersion:  "v1.2.3",
		TargetPath:     targetPath,
		ReleaseBaseURL: server.URL + "/releases",
		HTTPClient:     server.Client(),
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
}

func buildArchive(t *testing.T, binaryContents string) []byte {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), "skills.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	gzw := gzip.NewWriter(file)
	tw := tar.NewWriter(gzw)

	data := []byte(binaryContents)
	if err := tw.WriteHeader(&tar.Header{
		Name: "skills",
		Mode: 0o755,
		Size: int64(len(data)),
	}); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close() tar error = %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("Close() gzip error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() file error = %v", err)
	}

	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return archive
}

func runtimeArch() string {
	return runtime.GOARCH
}
