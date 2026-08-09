package version

// Version is the WatchTower build version. It is injected at build time via
// -ldflags "-X github.com/watch-tower-org/watchtower/backend/internal/version.Version=...".
// Defaults to "dev" when not injected.
var Version = "dev"
