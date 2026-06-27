# Himawari-8 Wallpaper

Live Earth desktop wallpaper from the [Himawari-8](https://himawari8.nict.go.jp/) weather satellite. A single Windows executable downloads the latest full-disk image and sets it as your wallpaper, optionally on a schedule.

Image data is provided by [NICT](https://www.nict.go.jp/) (Japan).

## Download

Get `himawari8.exe` from the [Releases](https://github.com/nikolaswheatten/himawari8-wallpaper/releases) page. No runtime dependencies — just download and run.

## Quick start

```cmd
himawari8 once
```

Downloads the latest image (2200×2200 by default) and sets it as your wallpaper.

To keep it updated automatically:

```cmd
himawari8 install
```

This adds a startup shortcut. The updater runs in the background and refreshes every 15 minutes when a new image is available.

## Commands

| Command | Description |
|---------|-------------|
| `himawari8 once` | Download and set wallpaper once |
| `himawari8 run` | Loop: check for updates every 15 minutes |
| `himawari8 install` | Add to Windows Startup (runs `run --quiet`) |
| `himawari8 uninstall` | Remove from Windows Startup |
| `himawari8 status` | Show whether autostart is enabled |

## Options

```cmd
himawari8 run --interval 10 --verbose
himawari8 once --width 275 --output D:\Pictures\earth.jpg
```

| Flag | Default | Description |
|------|---------|-------------|
| `--width` | `550` | Tile width; final size is width × 4 |
| `--interval` | `15` | Minutes between updates (`run` only) |
| `--output` | `wallpaper.jpg` | Output path (next to the exe by default) |
| `--verbose` | off | Print download progress |
| `--quiet` | off | Hide console window (`run` only) |

## Resolution guide

| `--width` | Level | Output size | Approx. file size |
|-----------|-------|-------------|-------------------|
| 110 | 1d | 440×440 | ~5 KB |
| 275 | 2d | 1100×1100 | ~30 KB |
| 550 | 4d | 2200×2200 | ~150 KB |
| 1100 | 8d | 4400×4400 | ~600 KB |
| 2200 | 20d | 8800×8800 | ~2.5 MB |

550 (2200×2200) is the recommended balance of quality and download size.

## Behavior

- **Skip unchanged**: `run` checks `latest.json` first; if the satellite timestamp matches the last successful download, tile downloads are skipped.
- **Single instance**: only one `run` process can be active at a time.
- **Autostart**: `install` creates a shortcut in your Startup folder with a hidden console window.

## Requirements

- Windows 10 or later
- Internet access

## Build from source

Requires [Go 1.22+](https://go.dev/).

```cmd
go build -ldflags="-s -w" -o himawari8.exe ./cmd/himawari8
```

## License

MIT — see [LICENSE](LICENSE). Himawari-8 imagery © NICT.
