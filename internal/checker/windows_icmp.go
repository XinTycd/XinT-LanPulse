//go:build windows

package checker

import (
	"context"
	"encoding/binary"
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

var icmpDLL = windows.NewLazySystemDLL("iphlpapi.dll")
var icmpCreateFile = icmpDLL.NewProc("IcmpCreateFile")
var icmpCloseHandle = icmpDLL.NewProc("IcmpCloseHandle")
var icmpSendEcho = icmpDLL.NewProc("IcmpSendEcho")

const ipSuccess = 0

type echoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       [16]byte
}
type WindowsICMP struct{}

func New() *WindowsICMP { return &WindowsICMP{} }
func (w *WindowsICMP) Ping(ctx context.Context, address string, timeoutMS, attempts int) Result {
	r := Result{Address: address, Attempts: attempts, TimeoutMS: timeoutMS}
	ip := net.ParseIP(address).To4()
	if ip == nil {
		return r
	}
	h, _, _ := icmpCreateFile.Call()
	if h == 0 || h == ^uintptr(0) {
		return r
	}
	defer icmpCloseHandle.Call(h)
	dst := binary.LittleEndian.Uint32(ip)
	payload := []byte("XinT-LanPulse")
	reply := make([]byte, int(unsafe.Sizeof(echoReply{}))+len(payload)+8)
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return r
		}
		ret, _, _ := icmpSendEcho.Call(h, uintptr(dst), uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)), 0, uintptr(unsafe.Pointer(&reply[0])), uintptr(len(reply)), uintptr(timeoutMS))
		if ret > 0 {
			er := (*echoReply)(unsafe.Pointer(&reply[0]))
			if er.Status == ipSuccess {
				r.Online = true
				r.LatencyMS = int(er.RoundTripTime)
				return r
			}
		}
	}
	return r
}
