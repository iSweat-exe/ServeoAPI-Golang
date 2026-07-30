//go:build windows

package files

import (
	"os"
)

// chownFileToMatchRoot sur Windows est une no-op car Windows utilise les ACLs
// et non pas le système simple d'UID/GID d'UNIX. Docker Desktop (WSL) gère
// automatiquement les permissions sur les montages de volumes.
func chownFileToMatchRoot(f *os.File, root *os.Root) {
	// Pas d'action requise sous Windows
}
