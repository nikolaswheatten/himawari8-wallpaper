//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/himawari8-wallpaper/himawari8/internal/autostart"
	"github.com/himawari8-wallpaper/himawari8/internal/himawari"
	"github.com/himawari8-wallpaper/himawari8/internal/instance"
	"github.com/himawari8-wallpaper/himawari8/internal/wallpaper"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	showWindow     = user32.NewProc("ShowWindow")
)

const swHide = 0

type sharedFlags struct {
	width    int
	output   string
	verbose  bool
	interval int
	quiet    bool
}

func exePaths() (exePath, exeDir string, err error) {
	exePath, err = os.Executable()
	if err != nil {
		return "", "", err
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return "", "", err
	}
	return exePath, filepath.Dir(exePath), nil
}

func hideConsole() {
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		showWindow.Call(hwnd, swHide)
	}
}

func makeConfig(exeDir string, f sharedFlags) himawari.Config {
	cfg := himawari.DefaultConfig(exeDir)
	cfg.Width = f.width
	cfg.Verbose = f.verbose
	if f.output != "" {
		cfg.OutputPath = f.output
	}
	return cfg
}

func updateWallpaper(cfg himawari.Config, force bool) error {
	result, err := cfg.DownloadAndSave(force)
	if err != nil {
		return err
	}

	if result.Skipped {
		if cfg.Verbose {
			fmt.Printf("Wallpaper already up to date (%s)\n", result.Date)
		}
		return wallpaper.Set(result.OutputPath)
	}

	if err := wallpaper.Set(result.OutputPath); err != nil {
		return fmt.Errorf("set wallpaper: %w", err)
	}

	if cfg.Verbose {
		fmt.Println("Wallpaper set successfully")
	}
	return nil
}

func cmdOnce(f sharedFlags) int {
	_, exeDir, err := exePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	cfg := makeConfig(exeDir, f)
	if cfg.Verbose {
		fmt.Println("=== Himawari-8 Wallpaper Updater ===")
		fmt.Printf("Resolution: %dx%d\n", cfg.ImageSize(), cfg.ImageSize())
		fmt.Printf("Output: %s\n", cfg.OutputPath)
	}

	if err := updateWallpaper(cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func cmdRun(f sharedFlags) int {
	if f.quiet {
		hideConsole()
	}

	lock, err := instance.Acquire()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	defer lock.Release()

	_, exeDir, err := exePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	cfg := makeConfig(exeDir, f)
	interval := time.Duration(f.interval) * time.Minute

	if cfg.Verbose {
		fmt.Println("=== Himawari-8 Wallpaper Updater ===")
		fmt.Printf("Update interval: %d minutes\n", f.interval)
		fmt.Printf("Resolution: %dx%d\n", cfg.ImageSize(), cfg.ImageSize())
		fmt.Printf("Output: %s\n", cfg.OutputPath)
		fmt.Println()
	}

	for {
		if cfg.Verbose {
			fmt.Println("Updating wallpaper...")
		}
		if err := updateWallpaper(cfg, false); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else if cfg.Verbose {
			fmt.Println("Wallpaper updated")
		}

		if cfg.Verbose {
			fmt.Printf("Next update in %d minutes...\n", f.interval)
		}
		time.Sleep(interval)
	}
}

func cmdInstall() int {
	exePath, exeDir, err := exePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if err := autostart.Install(exePath, exeDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Println("Autostart installed successfully.")
	fmt.Println("The updater will start automatically when Windows starts.")
	return 0
}

func cmdUninstall() int {
	enabled, path, err := autostart.IsEnabled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if !enabled {
		fmt.Println("Autostart is not enabled.")
		return 0
	}

	if err := autostart.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Println("Autostart removed successfully.")
	fmt.Printf("Removed: %s\n", path)
	return 0
}

func cmdStatus() int {
	enabled, path, err := autostart.IsEnabled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if enabled {
		fmt.Println("Autostart: ENABLED")
		fmt.Printf("Shortcut: %s\n", path)
		return 0
	}

	fmt.Println("Autostart: DISABLED")
	return 1
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Himawari-8 Wallpaper Updater

Usage:
  himawari8 run        Update wallpaper on an interval (default: every 15 minutes)
  himawari8 once       Download and set wallpaper once
  himawari8 install    Add to Windows Startup folder
  himawari8 uninstall  Remove from Windows Startup folder
  himawari8 status     Check autostart status

Common flags:
  --width N       Tile width (default: 550, output is width*4)
  --output PATH   Output JPEG path (default: wallpaper.jpg next to exe)
  --verbose       Verbose output

run flags:
  --interval N    Minutes between updates (default: 15)
  --quiet         Hide console window (used by autostart)

Examples:
  himawari8 once
  himawari8 run --interval 10 --verbose
  himawari8 install
`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		printUsage()
		return
	}

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	f := sharedFlags{}
	fs.IntVar(&f.width, "width", 550, "tile width in pixels")
	fs.StringVar(&f.output, "output", "", "output JPEG path")
	fs.BoolVar(&f.verbose, "verbose", false, "verbose output")
	fs.IntVar(&f.interval, "interval", 15, "minutes between updates")
	fs.BoolVar(&f.quiet, "quiet", false, "hide console window")

	switch cmd {
	case "once":
		_ = fs.Parse(os.Args[2:])
		os.Exit(cmdOnce(f))
	case "run":
		_ = fs.Parse(os.Args[2:])
		os.Exit(cmdRun(f))
	case "install":
		_ = fs.Parse(os.Args[2:])
		os.Exit(cmdInstall())
	case "uninstall":
		_ = fs.Parse(os.Args[2:])
		os.Exit(cmdUninstall())
	case "status":
		_ = fs.Parse(os.Args[2:])
		os.Exit(cmdStatus())
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}
