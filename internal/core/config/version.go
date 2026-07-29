package config

// These variables will be injected by the compiler via ldflags:
// go build -ldflags "-X serveoapi/internal/core/config.AppVersion=v2.0.0 -X serveoapi/internal/core/config.CommitHash=abc1234"
var (
	AppVersion = "dev"
	CommitHash = "none"
	DefaultAPIToken = "isweat_123" // Can be overridden by ldflags in CI/CD
)
