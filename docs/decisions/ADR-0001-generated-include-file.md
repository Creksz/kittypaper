# ADR-0001: Manage wallpaper through generated include file

## Status

Accepted

## Context

The project must not directly overwrite `kitty.conf`, but still needs to apply wallpaper updates safely.

## Decision

Kittypaper writes wallpaper configuration to a generated file (for example `kittypaper-background.conf`) and requires that file to be included from `kitty.conf`.

## Consequences

- Lower risk of corrupting user-maintained `kitty.conf`.
- Clear ownership boundary between user config and generated config.
- Reload operation can remain deterministic after each update.
