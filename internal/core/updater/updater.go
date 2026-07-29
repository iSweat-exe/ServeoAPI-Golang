package updater

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/minio/selfupdate"
	"serveoapi/internal/core/config"
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
	log.Printf("Checking for updates (Current version: %s)...", config.AppVersion)

	resp, err := http.Get(repoAPI)
	if err != nil {
		log.Fatalf("❌ Failed to contact GitHub API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("❌ GitHub API returned status: %s", resp.Status)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Fatalf("❌ Failed to parse GitHub API response: %v", err)
	}

	if release.TagName == config.AppVersion {
		log.Printf("ServeoAPI is already up-to-date (%s)", config.AppVersion)
		return
	}

	log.Printf("New version found: %s! Looking for compatible binary...", release.TagName)

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
		log.Fatalf("❌ No compatible binary found for %s in release %s", expectedName, release.TagName)
	}

	log.Printf("Downloading %s...", downloadURL)
	downloadResp, err := http.Get(downloadURL)
	if err != nil {
		log.Fatalf("❌ Failed to download update: %v", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		log.Fatalf("❌ Failed to download update, status: %s", downloadResp.Status)
	}

	log.Println("Applying update...")
	err = selfupdate.Apply(downloadResp.Body, selfupdate.Options{})
	if err != nil {
		log.Fatalf("❌ Update failed: %v", err)
	}

	log.Printf("🎉 Update successful! ServeoAPI is now at version %s.", release.TagName)
	log.Println("Restart the service to use the new version.")
}
