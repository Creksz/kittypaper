package version

// Set at build time via -ldflags.
var (
	Version = "0.1.0"
	Commit  = "dev"
)

func String() string {
	if Commit == "" || Commit == "dev" {
		return Version
	}
	return Version + " (" + Commit + ")"
}
