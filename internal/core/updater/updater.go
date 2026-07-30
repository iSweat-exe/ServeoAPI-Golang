package updater

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"

	"serveoapi/internal/core/config"

	"github.com/minio/selfupdate"
)

const repoAPI = "https://api.github.com/repos/iSweat-exe/ServeoAPI-Golang/releases/latest"

type GithubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GithubAsset `json:"assets"`
}

type GithubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// RunCheckAndUpdate checks for a new release and applies the update if one is found.
func RunCheckAndUpdate() {
	slog.Info("Checking for updates", "current_version", config.AppVersion)

	resp, err := http.Get(repoAPI)
	if err != nil {
		slog.Error("Failed to contact GitHub API", "error", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("GitHub API returned non-OK status", "status", resp.Status)
		os.Exit(1)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		slog.Error("Failed to parse GitHub API response", "error", err)
		os.Exit(1)
	}

	if release.TagName == config.AppVersion {
		slog.Info("ServeoAPI is already up-to-date", "version", config.AppVersion)
		return
	}

	slog.Info("New version found! Looking for compatible binary...", "version", release.TagName)

	// Build expected asset name, e.g. "serveoapi_linux_amd64"
	expectedName := fmt.Sprintf("serveoapi_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		expectedName += ".exe"
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if strings.Contains(strings.ToLower(asset.Name), expectedName) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		slog.Error(
			"No compatible binary found",
			"expected",
			expectedName,
			"release",
			release.TagName,
		)
		os.Exit(1)
	}

	slog.Info("Downloading update...", "url", downloadURL)
	downloadResp, err := http.Get(downloadURL)
	if err != nil {
		slog.Error("Failed to download update", "error", err)
		os.Exit(1)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		slog.Error("Failed to download update, bad status", "status", downloadResp.Status)
		os.Exit(1)
	}

	slog.Info("Applying update...")
	err = selfupdate.Apply(downloadResp.Body, selfupdate.Options{})
	if err != nil {
		slog.Error("Update failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Update successful!", "new_version", release.TagName)
	slog.Info("Restart the service to use the new version.")
}
