package contextkeys

import "context"

type ContextKey string

const (
	UserPermissionsKey ContextKey = "user_permissions"
	UserIDKey          ContextKey = "user_id"
)

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
