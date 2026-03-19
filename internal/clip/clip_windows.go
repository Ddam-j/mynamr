//go:build windows

package clip

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	procOpenClipboard     = user32.NewProc("OpenClipboard")
	procCloseClipboard    = user32.NewProc("CloseClipboard")
	procGetClipboardData  = user32.NewProc("GetClipboardData")
	procSetClipboardData  = user32.NewProc("SetClipboardData")
	procEmptyClipboard    = user32.NewProc("EmptyClipboard")
	procIsFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalAlloc       = kernel32.NewProc("GlobalAlloc")
	procGlobalFree        = kernel32.NewProc("GlobalFree")
	procGlobalLock        = kernel32.NewProc("GlobalLock")
	procGlobalUnlock      = kernel32.NewProc("GlobalUnlock")
)

const clipboardOpenRetryCount = 5
const clipboardOpenRetryDelay = 20 * time.Millisecond

type systemClipboard struct{}

func New() Interface {
	return systemClipboard{}
}

func (systemClipboard) ReadText() (string, error) {
	if err := openClipboard(); err != nil {
		return "", err
	}
	defer closeClipboard()

	available, _, err := procIsFormatAvailable.Call(uintptr(cfUnicodeText))
	if available == 0 {
		if err != windows.Errno(0) {
			return "", fmt.Errorf("check clipboard format: %w", err)
		}
		return "", fmt.Errorf("clipboard does not contain Unicode text")
	}

	handle, _, err := procGetClipboardData.Call(uintptr(cfUnicodeText))
	if handle == 0 {
		return "", fmt.Errorf("get clipboard data: %w", err)
	}

	locked, _, err := procGlobalLock.Call(handle)
	if locked == 0 {
		return "", fmt.Errorf("lock clipboard data: %w", err)
	}
	defer procGlobalUnlock.Call(handle)

	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(locked))), nil
}

func (systemClipboard) WriteText(text string) error {
	utf16, err := windows.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("encode clipboard text: %w", err)
	}

	size := uintptr(len(utf16) * 2)
	handle, _, err := procGlobalAlloc.Call(gmemMoveable, size)
	if handle == 0 {
		return fmt.Errorf("allocate clipboard buffer: %w", err)
	}

	locked, _, err := procGlobalLock.Call(handle)
	if locked == 0 {
		freeGlobal(handle)
		return fmt.Errorf("lock clipboard buffer: %w", err)
	}

	buffer := unsafe.Slice((*uint16)(unsafe.Pointer(locked)), len(utf16))
	copy(buffer, utf16)
	procGlobalUnlock.Call(handle)

	if err := openClipboard(); err != nil {
		freeGlobal(handle)
		return err
	}
	defer closeClipboard()

	empty, _, err := procEmptyClipboard.Call()
	if empty == 0 {
		freeGlobal(handle)
		return fmt.Errorf("empty clipboard: %w", err)
	}

	stored, _, err := procSetClipboardData.Call(uintptr(cfUnicodeText), handle)
	if stored == 0 {
		freeGlobal(handle)
		return fmt.Errorf("set clipboard data: %w", err)
	}
	return nil
}

func openClipboard() error {
	var lastErr error
	for i := 0; i < clipboardOpenRetryCount; i++ {
		opened, _, err := procOpenClipboard.Call(0)
		if opened != 0 {
			return nil
		}
		lastErr = fmt.Errorf("open clipboard: %w", err)
		time.Sleep(clipboardOpenRetryDelay)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("open clipboard: unknown failure")
	}
	return lastErr
}

func closeClipboard() {
	procCloseClipboard.Call()
}

func freeGlobal(handle uintptr) {
	if handle != 0 {
		procGlobalFree.Call(handle)
	}
}
