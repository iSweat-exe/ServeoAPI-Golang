// Signe un binaire de release avec minisign (Go), compatible avec le secret
// GitHub MINISIGN_PRIVATE_KEY même s'il ne contient que la ligne base64.
//
// Usage :
//
//	MINISIGN_PRIVATE_KEY="$(cat secrets/minisign.key)" go run ./scripts/sign-release -m serveoapi_linux_amd64
//	go run ./scripts/sign-release -key-file secrets/minisign.key -m serveoapi_linux_amd64 -x serveoapi_linux_amd64.minisig
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"aead.dev/minisign"
)

func main() {
	keyFile := flag.String(
		"key-file",
		"",
		"chemin du fichier clé privée (sinon env MINISIGN_PRIVATE_KEY)",
	)
	messagePath := flag.String("m", "", "fichier à signer")
	sigPath := flag.String("x", "", "fichier signature de sortie (défaut: <fichier>.minisig)")
	flag.Parse()

	if *messagePath == "" {
		fatalf("option -m requise")
	}
	if *sigPath == "" {
		*sigPath = *messagePath + ".minisig"
	}

	keyBytes, err := loadPrivateKeyBytes(*keyFile)
	if err != nil {
		fatalf("clé privée: %v", err)
	}

	privateKey, err := parsePrivateKey(keyBytes)
	if err != nil {
		fatalf("parse clé: %v", err)
	}

	message, err := os.ReadFile(*messagePath)
	if err != nil {
		fatalf("lecture %s: %v", *messagePath, err)
	}

	signature := minisign.Sign(privateKey, message)
	if err := os.WriteFile(*sigPath, signature, 0o644); err != nil {
		fatalf("écriture %s: %v", *sigPath, err)
	}

	fmt.Printf("signé %s -> %s (key_id=%X)\n", *messagePath, *sigPath, privateKey.ID())
}

func loadPrivateKeyBytes(keyFile string) ([]byte, error) {
	if keyFile != "" {
		return os.ReadFile(keyFile)
	}
	raw := strings.TrimSpace(os.Getenv("MINISIGN_PRIVATE_KEY"))
	if raw == "" {
		return nil, fmt.Errorf("fournir -key-file ou MINISIGN_PRIVATE_KEY")
	}
	return []byte(raw), nil
}

func parsePrivateKey(keyBytes []byte) (minisign.PrivateKey, error) {
	keyBytes = []byte(strings.ReplaceAll(string(keyBytes), "\r\n", "\n"))
	keyBytes = []byte(strings.TrimSpace(string(keyBytes)))

	if minisign.IsEncrypted(keyBytes) {
		password := os.Getenv("MINISIGN_PASSWORD")
		return minisign.DecryptKey(password, keyBytes)
	}

	var privateKey minisign.PrivateKey
	if err := privateKey.UnmarshalText(keyBytes); err != nil {
		return minisign.PrivateKey{}, err
	}
	return privateKey, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
