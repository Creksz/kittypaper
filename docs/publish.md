# Publish checklist

## Before first push

1. Replace `YOUR_USERNAME` in:
   - `README.md`
   - `CONTRIBUTING.md`
   - this file
2. Optional but recommended for `go install github.com/...`:
   - rename Go module in `go.mod` to `github.com/YOUR_USERNAME/kittypaper`
   - update imports, then `go mod tidy`
3. Create an empty GitHub repo named `kittypaper` (no README).

## Commit and push

```bash
cd ~/Project_Linux/kittypaper
git add .
git status
git commit -m "Initial release: Kitty wallpaper manager (CLI, TUI, GUI)"
git remote add origin git@github.com:YOUR_USERNAME/kittypaper.git
git push -u origin main
```

Or with GitHub CLI:

```bash
gh repo create kittypaper --public --source=. --remote=origin --push
```

## After publish

```bash
git tag v0.1.0
git push origin v0.1.0
```

Users can then install with:

```bash
git clone https://github.com/YOUR_USERNAME/kittypaper.git
cd kittypaper
make install
kittypaper gui
```

Verify GitHub Actions CI is green on the first push.
