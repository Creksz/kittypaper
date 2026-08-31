package kitty

type ReloadMethod string

const (
	ReloadAuto        ReloadMethod = "auto"
	ReloadKittyRemote ReloadMethod = "kitty-remote"
	ReloadSignal      ReloadMethod = "signal"
)
