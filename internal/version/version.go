// Package version holds the build version for the spekk binary.
// The version is set at build time via ldflags in main and stored here
// so other packages can access it (e.g., self-update, user-agent headers).
package version

// Version is the current build version. It defaults to "dev" and is
// overridden at build time via: go build -ldflags "-X main.version=1.2.3"
var Version = "dev"
