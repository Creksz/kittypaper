package kitty

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	domainerr "kittypaper/internal/domain/errors"
	domainkitty "kittypaper/internal/domain/kitty"
	"kittypaper/internal/platform/procx"
)

type Reloader struct {
	KittyBinary string
	Timeout     time.Duration
}

func (r Reloader) Reload(ctx context.Context, method domainkitty.ReloadMethod) error {
	if method == "" {
		method = domainkitty.ReloadAuto
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch method {
	case domainkitty.ReloadKittyRemote:
		return r.reloadRemote(ctx)
	case domainkitty.ReloadSignal:
		return procx.SignalKittyUSR1()
	case domainkitty.ReloadAuto:
		if err := r.reloadRemote(ctx); err == nil {
			return nil
		}
		if err := procx.SignalKittyUSR1(); err == nil {
			return nil
		} else {
			return fmt.Errorf("reload kitty: remote and signal failed: %w", err)
		}
	default:
		return fmt.Errorf("%w: %s", domainerr.ErrUnknownReloadMethod, method)
	}
}

func (r Reloader) ApplyImage(ctx context.Context, wallpaperPath string) error {
	_ = ctx
	_ = wallpaperPath
	// Config file is the source of truth for new terminals.
	// Reload running instances via signal (works for all kitty processes).
	return procx.SignalKittyUSR1()
}

func (r Reloader) reloadRemote(ctx context.Context) error {
	bin := r.KittyBinary
	if bin == "" {
		bin = "kitty"
	}
	return r.runKittyAt(ctx, bin, "@", "load-config")
}

func (r Reloader) runKittyAt(ctx context.Context, bin string, args ...string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	if sock := os.Getenv("KITTY_LISTEN_ON"); sock != "" {
		cmd.Env = append(os.Environ(), "KITTY_LISTEN_ON="+sock)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := bytes.TrimSpace(stderr.Bytes())
		if len(msg) > 0 {
			return fmt.Errorf("kitty %v: %s", args, msg)
		}
		return fmt.Errorf("kitty %v: %w", args, err)
	}
	return nil
}
