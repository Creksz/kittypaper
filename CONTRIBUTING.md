# Contributing to Kittypaper

Thanks for helping improve Kittypaper. Bug reports, docs, and pull requests are all welcome.

## Development setup

```bash
git clone https://github.com/Creksz/kittypaper.git
cd kittypaper
make build
make install   # installs kittypaper to ~/.local/bin
```

GUI builds need OpenGL/X11 libs (Arch):

```bash
sudo pacman -S gcc libxcursor libxrandr libxinerama libxi libglvnd mesa
```

## Architecture rules

- Keep business logic in `internal/app` and `internal/domain`.
- UI (`cli` / `tui` / `gui`) and Kitty/filesystem code live in `internal/adapters`.
- Prefer small, cohesive packages and composition.
- Do not overwrite user `kitty.conf` wallpaper keys; manage the generated include file.

See [docs/architecture.md](docs/architecture.md).

## Pull requests

1. Fork and create a branch.
2. Run `make lint` (fmt + vet).
3. Keep the change focused; update docs when behavior changes.
4. Open a PR with a short summary and test notes.

## Reporting issues

Include:

- Kitty version (`kitty --version`)
- OS / distro
- Command you ran (`kittypaper set ...`, `kittypaper gui`, etc.)
- Output of `kittypaper status`
- Relevant lines from `~/.config/kitty/kitty.conf` (especially `background_tint` and `include`)
