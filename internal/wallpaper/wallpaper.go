//go:build windows

package wallpaper

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	spiSetDeskWallpaper = 0x0014
	spifUpdateINIFile   = 0x01
	spifSendChange      = 0x02
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	systemParametersInfo = user32.NewProc("SystemParametersInfoW")
)

// Set sets the desktop wallpaper to the given image file path.
func Set(imagePath string) error {
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("resolve wallpaper path: %w", err)
	}

	pathUTF16, err := syscall.UTF16PtrFromString(absPath)
	if err != nil {
		return fmt.Errorf("encode wallpaper path: %w", err)
	}

	ret, _, callErr := systemParametersInfo.Call(
		uintptr(spiSetDeskWallpaper),
		0,
		uintptr(unsafe.Pointer(pathUTF16)),
		uintptr(spifUpdateINIFile|spifSendChange),
	)
	if ret == 0 {
		if callErr != nil && callErr.Error() != "The operation completed successfully." {
			return fmt.Errorf("SystemParametersInfoW: %w", callErr)
		}
		return fmt.Errorf("SystemParametersInfoW failed")
	}
	return nil
}
