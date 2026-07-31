// Générateur de paire minisign pour les releases ServeoAPI.
//
// Usage (depuis la racine ServeoAPI V2) :
//
//	go run ./scripts/generate-minisign-keys.go
//
// Écrit les fichiers dans ./secrets/ (gitignored) et affiche la clé publique
// à coller dans le secret GitHub MINISIGN_PUBLIC_KEY.
package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"aead.dev/minisign"
)

func main() {
	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		fatalf("génération: %v", err)
	}

	pubText, err := pub.MarshalText()
	if err != nil {
		fatalf("marshal public: %v", err)
	}
	privText, err := priv.MarshalText()
	if err != nil {
		fatalf("marshal private: %v", err)
	}

	outDir := filepath.Join("secrets")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		fatalf("mkdir: %v", err)
	}

	pubPath := filepath.Join(outDir, "minisign.pub")
	privPath := filepath.Join(outDir, "minisign.key")
	if err := os.WriteFile(pubPath, pubText, 0o600); err != nil {
		fatalf("écriture %s: %v", pubPath, err)
	}
	if err := os.WriteFile(privPath, privText, 0o600); err != nil {
		fatalf("écriture %s: %v", privPath, err)
	}

	rawPub := pub.String()
	fmt.Println("Paire minisign générée.")
	fmt.Println()
	fmt.Printf("  Fichier public  : %s\n", pubPath)
	fmt.Printf("  Fichier privé   : %s\n", privPath)
	fmt.Println()
	fmt.Println("Ajoutez ces secrets GitHub (Settings → Secrets and variables → Actions) :")
	fmt.Println()
	fmt.Println("  MINISIGN_PUBLIC_KEY  = (une seule ligne, ci-dessous)")
	fmt.Println(rawPub)
	fmt.Println()
	fmt.Println("  MINISIGN_PRIVATE_KEY = contenu COMPLET de secrets/minisign.key")
	fmt.Println("                         (y compris la ligne untrusted comment)")
	fmt.Println()
	fmt.Println("Ne committez jamais secrets/minisign.key.")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
