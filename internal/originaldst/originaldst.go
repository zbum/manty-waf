package originaldst

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// ParseSockaddr reads a Linux sockaddr written by SO_ORIGINAL_DST.
// The port is network byte order. The address family is native byte order.
func ParseSockaddr(raw []byte) (string, int, error) {
	if len(raw) < 8 {
		return "", 0, errors.New("원래 목적지 주소가 너무 짧습니다")
	}
	family := binary.NativeEndian.Uint16(raw[:2])
	port := int(binary.BigEndian.Uint16(raw[2:4]))
	switch family {
	case 2:
		return net.IP(raw[4:8]).String(), port, nil
	case 10:
		if len(raw) < 28 {
			return "", 0, errors.New("원래 목적지 주소가 너무 짧습니다")
		}
		return net.IP(raw[8:24]).String(), port, nil
	default:
		return "", 0, fmt.Errorf("지원하지 않는 주소 체계: %d", family)
	}
}
