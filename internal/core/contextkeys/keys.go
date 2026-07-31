package contextkeys

import "context"

type ContextKey string

const (
	UserPermissionsKey ContextKey = "user_permissions"
	UserIDKey          ContextKey = "user_id"
)

// SetUserID attaches the authenticated user ID to the context.
func SetUserID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

// SetUserPermissions attaches the comma-separated permission list to the context.
func SetUserPermissions(ctx context.Context, permissions string) context.Context {
	return context.WithValue(ctx, UserPermissionsKey, permissions)
}

// GetUserID securely retrieves the user ID from the context.
func GetUserID(ctx context.Context) (uint, bool) {
	id, ok := ctx.Value(UserIDKey).(uint)
	return id, ok
}

// GetUserPermissions securely retrieves the user permissions from the context.
func GetUserPermissions(ctx context.Context) (string, bool) {
	perms, ok := ctx.Value(UserPermissionsKey).(string)
	return perms, ok
}
