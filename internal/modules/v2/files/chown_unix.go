//go:build !windows

package files

import (
	"os"
	"syscall"
)

// chownFileToMatchRoot lit les permissions UID et GID du dossier racine (root)
// et les applique au fichier créé. Cela permet aux conteneurs de pouvoir
// modifier les fichiers sans avoir d'erreurs "Access Denied".
func chownFileToMatchRoot(f *os.File, root *os.Root) {
	rootInfo, err := root.Stat(".")
	if err != nil {
		return
	}

	if stat, ok := rootInfo.Sys().(*syscall.Stat_t); ok {
		_ = f.Chown(int(stat.Uid), int(stat.Gid))
	}
}
