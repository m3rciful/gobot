package buildinfo

// These variables are intended to be set via -ldflags at build time:
//
//	-X 'github.com/m3rciful/gobot/core/buildinfo.Version=v1.2.3'
//	-X 'github.com/m3rciful/gobot/core/buildinfo.Commit=abcdef0'
//	-X 'github.com/m3rciful/gobot/core/buildinfo.Date=2025-08-30T12:00:00Z'
//
// Default values are useful for local dev.
var (
	// Version reports the semantic version or tag of the build.
	Version = "dev"
	// Commit reports the source control commit used for the build.
	Commit = "local"
	// Date reports the build timestamp in RFC3339 format.
	Date = ""
)
