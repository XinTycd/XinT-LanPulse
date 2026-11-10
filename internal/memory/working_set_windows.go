//go:build windows

package memory

import (
	"golang.org/x/sys/windows"
)

var (
	psapi           = windows.NewLazySystemDLL("psapi.dll")
	emptyWorkingSet = psapi.NewProc("EmptyWorkingSet")
)

// TrimWorkingSet asks Windows to release cold physical pages. The Go heap and
// application state remain intact, so the next access is paged back in safely.
func TrimWorkingSet() {
	_, _, _ = emptyWorkingSet.Call(uintptr(windows.CurrentProcess()))
}
