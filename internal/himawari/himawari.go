package himawari

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ImageInfoURL = "https://himawari8.nict.go.jp/img/FULL_24h/latest.json"
	Level        = "4d"
	NumBlocks    = 4
	JPEGQuality  = 95
)

type latestInfo struct {
	Date string `json:"date"`
}

type Config struct {
	Width      int
	OutputPath string
	StatePath  string
	Verbose    bool
	HTTPClient *http.Client
}

type Result struct {
	Date      string
	Skipped   bool
	OutputPath string
}

func DefaultConfig(exeDir string) Config {
	return Config{
		Width:      550,
		OutputPath: filepath.Join(exeDir, "wallpaper.jpg"),
		StatePath:  filepath.Join(exeDir, ".himawari8-last"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c Config) ImageSize() int {
	return c.Width * NumBlocks
}

func (c Config) logf(format string, args ...any) {
	if c.Verbose {
		fmt.Printf(format+"\n", args...)
	}
}

func (c Config) FetchLatestDate() (string, error) {
	resp, err := c.HTTPClient.Get(ImageInfoURL)
	if err != nil {
		return "", fmt.Errorf("connect to Himawari-8 server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read latest.json: %w", err)
	}

	var info latestInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("parse latest.json: %w", err)
	}
	if info.Date == "" {
		return "", fmt.Errorf("latest.json missing date field")
	}

	if _, err := time.Parse("2006-01-02 15:04:05", info.Date); err != nil {
		return "", fmt.Errorf("parse date %q: %w", info.Date, err)
	}

	return info.Date, nil
}

func readLastDate(statePath string) (string, error) {
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeLastDate(statePath, date string) error {
	return os.WriteFile(statePath, []byte(date), 0644)
}

func levelForWidth(width int) string {
	switch width {
	case 110:
		return "1d"
	case 275:
		return "2d"
	case 550:
		return "4d"
	case 1100:
		return "8d"
	case 2200:
		return "20d"
	default:
		return Level
	}
}

func (c Config) DownloadAndSave(force bool) (Result, error) {
	date, err := c.FetchLatestDate()
	if err != nil {
		return Result{}, err
	}

	lastDate, err := readLastDate(c.StatePath)
	if err != nil {
		return Result{}, fmt.Errorf("read state file: %w", err)
	}

	if !force && lastDate == date {
		if _, err := os.Stat(c.OutputPath); err == nil {
			c.logf("Image unchanged (%s), skipping download", date)
			return Result{Date: date, Skipped: true, OutputPath: c.OutputPath}, nil
		}
	}

	c.logf("Latest image: %s", date)

	ts, err := time.Parse("2006-01-02 15:04:05", date)
	if err != nil {
		return Result{}, err
	}

	level := levelForWidth(c.Width)
	baseURL := fmt.Sprintf("https://himawari8.nict.go.jp/img/D531106/%s/%d", level, c.Width)
	imageURL := fmt.Sprintf("%s/%s/%s/%s/%s",
		baseURL,
		ts.Format("2006"),
		ts.Format("01"),
		ts.Format("02"),
		ts.Format("150405"),
	)

	imageSize := c.ImageSize()
	canvas := image.NewRGBA(image.Rect(0, 0, imageSize, imageSize))
	totalTiles := NumBlocks * NumBlocks
	downloaded := 0

	c.logf("Downloading image (%d tiles)...", totalTiles)

	tileClient := &http.Client{Timeout: 10 * time.Second}
	for y := 0; y < NumBlocks; y++ {
		for x := 0; x < NumBlocks; x++ {
			tileURL := fmt.Sprintf("%s_%d_%d.png", imageURL, x, y)
			tile, err := downloadTile(tileClient, tileURL)
			if err != nil {
				c.logf("  Warning: tile %d,%d: %v", x, y, err)
				continue
			}

			draw.Draw(canvas, image.Rect(x*c.Width, y*c.Width, (x+1)*c.Width, (y+1)*c.Width), tile, tile.Bounds().Min, draw.Src)
			downloaded++
			c.logf("  Downloaded tile %d/%d", downloaded, totalTiles)
		}
	}

	if downloaded == 0 {
		return Result{}, fmt.Errorf("failed to download any tiles")
	}

	c.logf("Downloaded tiles: %d/%d", downloaded, totalTiles)

	if err := saveJPEG(c.OutputPath, canvas); err != nil {
		return Result{}, err
	}

	if err := writeLastDate(c.StatePath, date); err != nil {
		return Result{}, fmt.Errorf("write state file: %w", err)
	}

	c.logf("Image saved: %s", c.OutputPath)
	return Result{Date: date, Skipped: false, OutputPath: c.OutputPath}, nil
}

func downloadTile(client *http.Client, url string) (image.Image, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return img, nil
}

func saveJPEG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return fmt.Errorf("encode JPEG: %w", err)
	}
	return nil
}
