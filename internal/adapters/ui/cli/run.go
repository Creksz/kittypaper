package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"kittypaper/internal/adapters/infra/config"
	"kittypaper/internal/adapters/ui/gui"
	"kittypaper/internal/adapters/ui/tui"
	"kittypaper/internal/app/dto"
	"kittypaper/internal/bootstrap"
	"kittypaper/internal/domain/kitty"
	"kittypaper/internal/version"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if err := run(args, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("kittypaper", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "path to kittypaper config.yaml")
	reloadMethod := fs.String("reload", "", "reload method: auto, kitty-remote, signal")
	fs.Usage = func() {
		fmt.Fprintf(stderr, `kittypaper — wallpaper manager for Kitty

Usage:
  kittypaper [flags] <command>

Commands:
  set PATH     apply a wallpaper file
  random       apply a random wallpaper from configured directories
  status       show active wallpaper and include status
  init         create config files and wire Kitty (first-time setup)
  setup        same as init + create ~/.config/kittypaper/config.yaml if missing
  restore      re-apply the last saved wallpaper (useful after reboot)
  tui          open interactive wallpaper picker (terminal)
  gui          open wallpaper browser window
  version      show version information

Flags:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	settings, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *reloadMethod != "" {
		settings.ReloadMethod = kitty.ReloadMethod(*reloadMethod)
	}

	container := bootstrap.NewContainer(settings)
	cmdArgs := fs.Args()
	if len(cmdArgs) == 0 {
		fs.Usage()
		return fmt.Errorf("missing command")
	}

	ctx := context.Background()
	switch cmdArgs[0] {
	case "set":
		if len(cmdArgs) < 2 {
			return fmt.Errorf("usage: kittypaper set PATH")
		}
		result, err := container.WallpaperService.SetWallpaper(ctx, dto.SetWallpaperRequest{
			Path:         strings.Join(cmdArgs[1:], " "),
			ReloadMethod: settings.ReloadMethod,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "applied %s\n", result.WallpaperPath)
		if result.Warning != "" {
			fmt.Fprintf(stderr, "warning: %s\n", result.Warning)
		}
		return nil
	case "random":
		result, err := container.WallpaperService.SetRandom(ctx, settings.ReloadMethod)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "applied %s\n", result.WallpaperPath)
		if result.Warning != "" {
			fmt.Fprintf(stderr, "warning: %s\n", result.Warning)
		}
		return nil
	case "status":
		status, err := container.WallpaperService.Status(ctx)
		if err != nil {
			return err
		}
		active := status.ActivePath
		if active == "" {
			active = "(none)"
		}
		fmt.Fprintf(stdout, "active: %s\n", active)
		fmt.Fprintf(stdout, "include: %t\n", status.IncludeOK)
		fmt.Fprintf(stdout, "kitty.conf: %s\n", status.KittyConfPath)
		fmt.Fprintf(stdout, "generated: %s\n", status.GeneratedConfPath)
		fmt.Fprintf(stdout, "wallpaper total: %d\n", status.WallpaperCount)
		return nil
	case "setup":
		result, err := bootstrap.Setup(ctx, *configPath)
		if err != nil {
			return err
		}
		if result.ConfigCreated {
			fmt.Fprintf(stdout, "created config: %s\n", result.ConfigPath)
		} else {
			fmt.Fprintf(stdout, "config: %s\n", result.ConfigPath)
		}
		fmt.Fprintf(stdout, "generated: %s\n", result.GeneratedConfPath)
		if result.IncludeWritten {
			fmt.Fprintf(stdout, "include added to %s\n", result.KittyConfPath)
		}
		fmt.Fprintf(stdout, "wallpaper folders:\n")
		for _, dir := range result.WallpaperDirs {
			fmt.Fprintf(stdout, "  - %s\n", dir)
		}
		fmt.Fprintf(stdout, "\nNext: put images in a wallpaper folder, then run: kittypaper gui\n")
		return nil
	case "init":
		writeInclude := true
		if len(cmdArgs) > 1 && cmdArgs[1] == "--no-write-include" {
			writeInclude = false
		}
		result, err := container.WallpaperService.Init(ctx, dto.InitRequest{WriteInclude: writeInclude})
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "generated: %s\n", result.GeneratedConfPath)
		if result.IncludeWritten {
			fmt.Fprintf(stdout, "include added to %s\n", result.KittyConfPath)
		} else {
			fmt.Fprintf(stdout, "include not modified (%s)\n", result.KittyConfPath)
		}
		if result.RestoredPath != "" {
			fmt.Fprintf(stdout, "restored %s\n", result.RestoredPath)
		}
		return nil
	case "restore":
		result, err := container.WallpaperService.Restore(ctx, settings.ReloadMethod)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "restored %s\n", result.WallpaperPath)
		if result.Warning != "" {
			fmt.Fprintf(stderr, "warning: %s\n", result.Warning)
		}
		return nil
	case "tui":
		return tui.Run(container.WallpaperService, settings.ReloadMethod)
	case "gui":
		return gui.Run(container)
	case "version":
		out := version.String()
		if len(cmdArgs) > 1 && (cmdArgs[1] == "--full" || cmdArgs[1] == "-v") {
			out = version.Full()
		}
		fmt.Fprintf(stdout, "kittypaper %s\n", out)
		return nil
	default:
		return fmt.Errorf("unknown command %q", cmdArgs[0])
	}
}

func Main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
