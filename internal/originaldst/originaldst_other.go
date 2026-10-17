//go:build !linux

package originaldst

import (
	"fmt"
	"net"
)

// Supported reports that SO_ORIGINAL_DST is unavailable on this operating system.
func Supported() bool { return false }

// Detect has no kernel original-destination lookup outside Linux.
func Detect(net.Conn) (string, int, error) {
	return "", 0, fmt.Errorf("투명 브리지는 리눅스의 SO_ORIGINAL_DST가 필요합니다")
}
