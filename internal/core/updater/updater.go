package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"serveoapi/internal/core/config"

	"github.com/minio/selfupdate"
)

const repoAPI = "https://api.github.com/repos/iSweat-exe/ServeoAPI-Golang/releases/latest"

// signatureSuffix est l'extension du fichier de signature minisign publié à côté du binaire.
const signatureSuffix = ".minisig"

var httpClient = &http.Client{Timeout: 5 * time.Minute}

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
	if err := checkAndUpdate(); err != nil {
		slog.Error("Update failed", "error", err)
		os.Exit(1)
	}
}

func checkAndUpdate() error {
	slog.Info("Checking for updates", "current_version", config.AppVersion)

	publicKey := updatePublicKey()
	if publicKey == "" {
		return errors.New(
			"aucune clé publique de mise à jour configurée : définissez SERVEO_UPDATE_PUBLIC_KEY " +
				"(ou injectez config.UpdatePublicKey au build) pour vérifier la signature du binaire",
		)
	}

	release, err := fetchLatestRelease()
	if err != nil {
		return err
	}

	if release.TagName == config.AppVersion {
		slog.Info("ServeoAPI is already up-to-date", "version", config.AppVersion)
		return nil
	}

	slog.Info("New version found! Looking for compatible binary...", "version", release.TagName)

	// Build expected asset name, e.g. "serveoapi_linux_amd64"
	expectedName := fmt.Sprintf("serveoapi_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		expectedName += ".exe"
	}

	binaryURL, signatureURL := findAssets(release, expectedName)
	if binaryURL == "" {
		return fmt.Errorf(
			"aucun binaire compatible trouvé (attendu: %s, release: %s)",
			expectedName,
			release.TagName,
		)
	}
	if signatureURL == "" {
		return fmt.Errorf(
			"aucune signature %s publiée pour %s : mise à jour refusée",
			signatureSuffix,
			expectedName,
		)
	}

	verifier := selfupdate.NewVerifier()
	if err := verifier.LoadFromURL(signatureURL, publicKey, http.DefaultTransport); err != nil {
		return fmt.Errorf("signature illisible ou clé publique invalide: %w", err)
	}

	slog.Info("Downloading update...", "url", binaryURL)
	downloadResp, err := httpClient.Get(binaryURL)
	if err != nil {
		return fmt.Errorf("téléchargement impossible: %w", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		return fmt.Errorf("téléchargement impossible, statut %s", downloadResp.Status)
	}

	slog.Info("Applying update...", "signature", signatureURL)
	if err := selfupdate.Apply(downloadResp.Body, selfupdate.Options{Verifier: verifier}); err != nil {
		if rollbackErr := selfupdate.RollbackError(err); rollbackErr != nil {
			slog.Error("Rollback de la mise à jour impossible", "error", rollbackErr)
		}
		return err
	}

	slog.Info("Update successful!", "new_version", release.TagName)
	slog.Info("Restart the service to use the new version.")
	return nil
}

// updatePublicKey retourne la clé publique minisign, injectée au build ou fournie par l'environnement.
func updatePublicKey() string {
	if key := strings.TrimSpace(os.Getenv("SERVEO_UPDATE_PUBLIC_KEY")); key != "" {
		return key
	}
	return strings.TrimSpace(config.UpdatePublicKey)
}

func fetchLatestRelease() (*GithubRelease, error) {
	resp, err := httpClient.Get(repoAPI)
	if err != nil {
		return nil, fmt.Errorf("contact de l'API GitHub impossible: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("l'API GitHub a répondu %s", resp.Status)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("réponse GitHub illisible: %w", err)
	}

	return &release, nil
}

// findAssets retourne l'URL du binaire correspondant à la plateforme et celle de sa signature.
func findAssets(release *GithubRelease, expectedName string) (binaryURL, signatureURL string) {
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		switch {
		case name == strings.ToLower(expectedName)+signatureSuffix:
			signatureURL = asset.BrowserDownloadURL
		case name == strings.ToLower(expectedName):
			binaryURL = asset.BrowserDownloadURL
		}
	}
	return binaryURL, signatureURL
}
