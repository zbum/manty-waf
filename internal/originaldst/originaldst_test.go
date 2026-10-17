package originaldst

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestParseSockaddrIPv4(t *testing.T) {
	raw := make([]byte, 16)
	binary.NativeEndian.PutUint16(raw[:2], 2)
	binary.BigEndian.PutUint16(raw[2:4], 8081)
	copy(raw[4:8], net.ParseIP("10.1.2.3").To4())

	host, port, err := ParseSockaddr(raw)
	if err != nil {
		t.Fatal(err)
	}
	if host != "10.1.2.3" || port != 8081 {
		t.Fatalf("host=%s port=%d", host, port)
	}
}

func TestParseSockaddrRejectsUnknownFamily(t *testing.T) {
	raw := make([]byte, 8)
	if _, _, err := ParseSockaddr(raw); err == nil {
		t.Fatal("empty family was accepted")
	}
}
