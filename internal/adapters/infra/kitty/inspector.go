package kitty

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	domainerr "kittypaper/internal/domain/errors"
	"kittypaper/internal/platform/filex"
	"kittypaper/internal/platform/pathx"
)

type Inspector struct {
	KittyConfPath     string
	GeneratedConfPath string
}

func (i Inspector) EnsureInclude(ctx context.Context) error {
	_ = ctx
	if _, err := os.Stat(i.KittyConfPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", domainerr.ErrKittyConfMissing, i.KittyConfPath)
		}
		return err
	}
	if err := RepairKittyConf(i.KittyConfPath, i.GeneratedConfPath); err != nil {
		return err
	}
	ok, err := HasInclude(i.KittyConfPath, i.GeneratedConfPath)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: add `include %s` to %s or run `kittypaper init`",
			domainerr.ErrKittyIncludeMissing,
			i.GeneratedConfPath,
			i.KittyConfPath,
		)
	}
	return nil
}

// RepairKittyConf fixes common issues that prevent wallpaper from loading on new terminals:
// - removes shared listen_on (breaks when multiple kitty instances start)
// - upgrades relative include to absolute path
func RepairKittyConf(kittyConfPath, generatedConfPath string) error {
	raw, err := os.ReadFile(kittyConfPath)
	if err != nil {
		return err
	}

	generatedAbs, err := filepath.Abs(generatedConfPath)
	if err != nil {
		generatedAbs = generatedConfPath
	}
	kittyDir := filepath.Dir(kittyConfPath)

	lines := strings.Split(string(raw), "\n")
	changed := false
	hasInclude := false
	var out []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}

		key, rest, ok := strings.Cut(trimmed, " ")
		if !ok {
			out = append(out, line)
			continue
		}

		switch key {
		case "listen_on":
			// Shared listen_on breaks additional kitty instances after reboot.
			changed = true
			continue
		case "include":
			target := strings.Trim(strings.TrimSpace(rest), `"'`)
			if includeMatches(target, kittyDir, generatedAbs) {
				hasInclude = true
				if target != generatedAbs {
					out = append(out, fmt.Sprintf("include %s", generatedAbs))
					changed = true
					continue
				}
			}
		}
		out = append(out, line)
	}

	if !hasInclude {
		block := includeBlock(generatedAbs)
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, strings.Split(strings.TrimSpace(block), "\n")...)
		changed = true
	}

	if !changed {
		return nil
	}

	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return filex.WriteFileAtomic(kittyConfPath, []byte(content), 0o644)
}

func HasInclude(kittyConfPath, generatedConfPath string) (bool, error) {
	file, err := os.Open(kittyConfPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	generatedAbs, err := filepath.Abs(generatedConfPath)
	if err != nil {
		generatedAbs = generatedConfPath
	}
	kittyDir := filepath.Dir(kittyConfPath)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, rest, ok := strings.Cut(line, " ")
		if !ok || key != "include" {
			continue
		}
		target := strings.Trim(strings.TrimSpace(rest), `"'`)
		if includeMatches(target, kittyDir, generatedAbs) {
			return true, nil
		}
	}
	return false, scanner.Err()
}

func AppendInclude(kittyConfPath, generatedConfPath string) error {
	generatedAbs, err := filepath.Abs(generatedConfPath)
	if err != nil {
		return err
	}
	return RepairKittyConf(kittyConfPath, generatedAbs)
}

func includeBlock(generatedAbs string) string {
	return fmt.Sprintf("# Managed by kittypaper — wallpaper include (must be last)\ninclude %s\n", generatedAbs)
}

func includeMatches(target, kittyConfDir, generatedAbs string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	target = pathx.Expand(target)
	resolved := target
	if !filepath.IsAbs(target) {
		resolved = filepath.Join(kittyConfDir, target)
	}
	if filepath.Clean(resolved) == filepath.Clean(generatedAbs) {
		return true
	}
	return filepath.Base(target) == filepath.Base(generatedAbs)
}
