package mcp

import (
	"context"
	"testing"

	"serveoapi/internal/core/contextkeys"
)

// Le middleware JWT injecte les permissions sous forme de chaîne : le serveur MCP
// castait auparavant en []string, ce qui faisait échouer tous les outils en production.
func TestHasPermissionReadsStringFromContext(t *testing.T) {
	ctx := contextkeys.SetUserPermissions(
		context.Background(),
		"docker.containers.read,system.read",
	)

	if err := hasPermission(ctx, "docker.containers.read"); err != nil {
		t.Fatalf("expected the permission to be granted, got %v", err)
	}
	if err := hasPermission(ctx, "users.manage"); err == nil {
		t.Fatal("expected a missing permission to be rejected")
	}
}

func TestHasPermissionWithoutContextValue(t *testing.T) {
	if err := hasPermission(context.Background(), "system.read"); err == nil {
		t.Fatal("expected an unauthenticated context to be rejected")
	}
}

func TestHasPermissionWithRootWildcard(t *testing.T) {
	ctx := contextkeys.SetUserPermissions(context.Background(), "*")

	if err := hasPermission(ctx, "docker.containers.delete"); err != nil {
		t.Fatalf("root permission should grant everything, got %v", err)
	}
}
