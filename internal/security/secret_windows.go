//go:build windows

package security

import (
	"encoding/base64"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var crypt32 = windows.NewLazySystemDLL("crypt32.dll")
var cryptProtectData = crypt32.NewProc("CryptProtectData")
var cryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
var localFree = windows.NewLazySystemDLL("kernel32.dll").NewProc("LocalFree")

func blob(b []byte) dataBlob {
	if len(b) == 0 {
		return dataBlob{}
	}
	return dataBlob{uint32(len(b)), &b[0]}
}
func Encrypt(plain string) (string, error) {
	in := []byte(plain)
	input, output := blob(in), dataBlob{}
	r, _, e := cryptProtectData.Call(uintptr(unsafe.Pointer(&input)), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&output)))
	if r == 0 {
		return "", fmt.Errorf("DPAPI 加密失败: %w", e)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(output.pbData)))
	return base64.StdEncoding.EncodeToString(unsafe.Slice(output.pbData, output.cbData)), nil
}
func Decrypt(cipher string) (string, error) {
	if cipher == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(cipher)
	if err != nil {
		return "", err
	}
	input, output := blob(raw), dataBlob{}
	r, _, e := cryptUnprotectData.Call(uintptr(unsafe.Pointer(&input)), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&output)))
	if r == 0 {
		return "", fmt.Errorf("DPAPI 解密失败: %w", e)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(output.pbData)))
	return string(unsafe.Slice(output.pbData, output.cbData)), nil
}
