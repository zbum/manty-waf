//go:build linux

package originaldst

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

const soOriginalDst = 80

// Supported reports that this process can ask the kernel for the original destination.
func Supported() bool { return true }

// Detect reads the destination from before an iptables REDIRECT or TPROXY.
func Detect(conn net.Conn) (string, int, error) {
	tcp, ok := conn.(*net.TCPConn)
	if !ok {
		return "", 0, fmt.Errorf("투명 브리지는 TCP 연결만 지원합니다")
	}
	raw, err := tcp.SyscallConn()
	if err != nil {
		return "", 0, err
	}
	var host string
	var port int
	var detectErr error
	ctrlErr := raw.Control(func(fd uintptr) {
		buf, sockErr := getsockopt(fd, syscall.IPPROTO_IP)
		if sockErr != nil {
			buf, sockErr = getsockopt(fd, syscall.IPPROTO_IPV6)
		}
		if sockErr != nil {
			detectErr = sockErr
			return
		}
		host, port, detectErr = ParseSockaddr(buf)
	})
	if ctrlErr != nil {
		return "", 0, ctrlErr
	}
	if detectErr != nil {
		return "", 0, detectErr
	}
	return host, port, nil
}

func getsockopt(fd uintptr, proto int) ([]byte, error) {
	buf := make([]byte, 128)
	length := uint32(len(buf))
	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		fd,
		uintptr(proto),
		soOriginalDst,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&length)),
		0,
	)
	if errno != 0 {
		return nil, errno
	}
	if length > uint32(len(buf)) {
		length = uint32(len(buf))
	}
	return buf[:length], nil
}
