package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
)

// VerifyChecksum checks that sha256(data) matches the entry for name in a
// checksums.txt body (the output of `sha256sum *.tar.gz`).
func VerifyChecksum(checksums []byte, name string, data []byte) error {
	var want string
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("checksum: no entry for %q", name)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %q: got %s, want %s", name, got, want)
	}
	return nil
}

// ExtractBinary gunzips and untars tarball and returns the bytes of the entry
// whose base name is "blvckhole".
func ExtractBinary(tarball []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(tarball))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if path.Base(hdr.Name) == "blvckhole" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("no blvckhole binary found in archive")
}
