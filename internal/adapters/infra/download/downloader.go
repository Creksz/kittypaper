package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)


type ProgressFunc func(done, total int64)

type Downloader struct {
	HTTP *http.Client
}

func (d Downloader) client() *http.Client {
	if d.HTTP != nil {
		return d.HTTP
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

func (d Downloader) Download(ctx context.Context, sourceURL, destDir, filename string, onProgress ProgressFunc) (string, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return "", fmt.Errorf("download URL is empty")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	if filename == "" {
		filename = filepath.Base(strings.Split(sourceURL, "?")[0])
	}
	if filename == "" || filename == "." {
		filename = fmt.Sprintf("wallpaper-%d.jpg", time.Now().Unix())
	}
	destPath := filepath.Join(destDir, filename)
	if _, err := os.Stat(destPath); err == nil {
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		ext := filepath.Ext(filename)
		destPath = filepath.Join(destDir, fmt.Sprintf("%s-%d%s", base, time.Now().Unix(), ext))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "kittypaper/0.1")

	resp, err := d.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	tmpPath := destPath + ".part"
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			out.Close()
			os.Remove(tmpPath)
			return "", ctx.Err()
		default:
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				out.Close()
				os.Remove(tmpPath)
				return "", err
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(written, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			out.Close()
			os.Remove(tmpPath)
			return "", readErr
		}
	}
	if err := out.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", err
	}
	_ = os.Chmod(destPath, 0o644)
	return destPath, nil
}
