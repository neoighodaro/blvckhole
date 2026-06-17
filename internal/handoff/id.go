package handoff

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"time"
)

// NewID returns a sortable, unique id: 8 bytes of UnixNano (big-endian) plus
// 4 random bytes, hex-encoded (24 chars). Sortability is best-effort; the
// random suffix guarantees uniqueness.
func NewID() string {
	var buf [12]byte
	binary.BigEndian.PutUint64(buf[0:8], uint64(time.Now().UTC().UnixNano()))
	_, _ = rand.Read(buf[8:12])
	return hex.EncodeToString(buf[:])
}
