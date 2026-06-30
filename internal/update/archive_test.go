package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: name, Mode: 0755, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("the binary tarball bytes")
	sum := sha256.Sum256(data)
	line := fmt.Sprintf("%s  blvckhole-v0.0.5-darwin-arm64.tar.gz\n", hex.EncodeToString(sum[:]))
	checksums := []byte("deadbeef  other.tar.gz\n" + line)

	if err := VerifyChecksum(checksums, "blvckhole-v0.0.5-darwin-arm64.tar.gz", data); err != nil {
		t.Errorf("VerifyChecksum (valid) = %v, want nil", err)
	}
	if err := VerifyChecksum(checksums, "blvckhole-v0.0.5-darwin-arm64.tar.gz", []byte("tampered")); err == nil {
		t.Error("VerifyChecksum should fail on mismatched data")
	}
	if err := VerifyChecksum(checksums, "missing.tar.gz", data); err == nil {
		t.Error("VerifyChecksum should fail when name is absent")
	}
}

// makeDirTarGz creates a tar.gz containing a single directory entry named name.
func makeDirTarGz(t *testing.T, name string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Typeflag: tar.TypeDir,
		Name:     name,
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestExtractBinary(t *testing.T) {
	want := []byte("#!/fake/elf/binary")
	tgz := makeTarGz(t, "blvckhole", want)
	got, err := ExtractBinary(tgz)
	if err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("ExtractBinary = %q, want %q", got, want)
	}

	if _, err := ExtractBinary(makeTarGz(t, "notblvckhole", want)); err == nil {
		t.Error("ExtractBinary should fail when no blvckhole entry exists")
	}

	// A directory entry named "blvckhole" must NOT be matched — only regular files.
	if _, err := ExtractBinary(makeDirTarGz(t, "blvckhole/")); err == nil {
		t.Error("ExtractBinary should fail when the only 'blvckhole' entry is a directory")
	}
}
