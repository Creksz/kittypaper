package version

// Set at build time via -ldflags.
var (
	Version = "0.2.0"
	Commit  = "dev"
)

// String returns the user-facing version (no git hash).
func String() string {
	return Version
}

// Full returns version with commit, for debugging only.
func Full() string {
	if Commit == "" || Commit == "dev" {
		return Version
	}
	return Version + "+" + Commit
}
