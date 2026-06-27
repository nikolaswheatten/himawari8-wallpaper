//go:build windows

package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

const shortcutName = "Himawari8WallpaperUpdater.lnk"

func startupFolder() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("APPDATA environment variable is not set")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup"), nil
}

func shortcutPath() (string, error) {
	folder, err := startupFolder()
	if err != nil {
		return "", err
	}
	return filepath.Join(folder, shortcutName), nil
}

func IsEnabled() (bool, string, error) {
	path, err := shortcutPath()
	if err != nil {
		return false, "", err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, path, nil
	}
	if err != nil {
		return false, path, err
	}
	return true, path, nil
}

func Install(exePath, workingDir string) error {
	shortcut, err := shortcutPath()
	if err != nil {
		return err
	}

	if err := os.Remove(shortcut); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing shortcut: %w", err)
	}

	if err := ole.CoInitialize(0); err != nil {
		return fmt.Errorf("initialize COM: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return fmt.Errorf("create WScript.Shell: %w", err)
	}
	defer unknown.Release()

	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("query WScript.Shell: %w", err)
	}
	defer shell.Release()

	linkVar, err := oleutil.CallMethod(shell, "CreateShortcut", shortcut)
	if err != nil {
		return fmt.Errorf("CreateShortcut: %w", err)
	}
	link := linkVar.ToIDispatch()
	defer link.Release()

	if _, err := oleutil.PutProperty(link, "TargetPath", exePath); err != nil {
		return fmt.Errorf("set TargetPath: %w", err)
	}
	if _, err := oleutil.PutProperty(link, "Arguments", "run --quiet"); err != nil {
		return fmt.Errorf("set Arguments: %w", err)
	}
	if _, err := oleutil.PutProperty(link, "WorkingDirectory", workingDir); err != nil {
		return fmt.Errorf("set WorkingDirectory: %w", err)
	}
	if _, err := oleutil.PutProperty(link, "Description", "Himawari-8 Wallpaper Updater"); err != nil {
		return fmt.Errorf("set Description: %w", err)
	}
	if _, err := oleutil.PutProperty(link, "WindowStyle", 7); err != nil {
		return fmt.Errorf("set WindowStyle: %w", err)
	}
	if _, err := oleutil.CallMethod(link, "Save"); err != nil {
		return fmt.Errorf("save shortcut: %w", err)
	}

	return nil
}

func Uninstall() error {
	shortcut, err := shortcutPath()
	if err != nil {
		return err
	}

	if err := os.Remove(shortcut); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove shortcut: %w", err)
	}
	return nil
}
