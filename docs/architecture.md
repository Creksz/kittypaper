# Kittypaper Architecture (v2)

## Goals

- One shared core for CLI, TUI, and GUI.
- No direct overwrite of `kitty.conf`.
- Generate and manage a dedicated include file for wallpaper settings.
- Keep business logic independent from UI and infrastructure.

## Layering

- `domain`: entities and core rules.
- `app`: use-cases and orchestration.
- `ports/outbound`: infrastructure contracts used by `app`.
- `adapters/ui`: CLI/TUI/GUI adapters that call `app`.
- `adapters/infra`: filesystem, kitty reload, config, cache implementations.
- `bootstrap`: dependency wiring entrypoint.

## Dependency Direction

`cmd -> adapters/ui -> app -> ports/outbound <- adapters/infra`

## First Implementations

Done:

- Kitty include inspector (`set` never overwrites `kitty.conf`)
- Generated `kittypaper-background.conf` atomic writer
- Reload strategy: `kitty @ load-config`, then SIGUSR1
- Filesystem wallpaper index + JSON active state
- CLI commands: `init`, `set`, `random`, `status`, `tui`, `version`
- TUI binary: `kittypaper-tui` (Bubble Tea list picker)

- Native desktop GUI (Fyne wallpaper browser)
