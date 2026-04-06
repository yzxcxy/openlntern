package database

import (
	"context"
	"strings"
)

type openVikingTenantContextKey string

const (
	openVikingAccountContextKey openVikingTenantContextKey = "openviking_account_id"
	openVikingUserContextKey    openVikingTenantContextKey = "openviking_user_id"
)

// WithOpenVikingTenant stores the tenant headers required by root-key OpenViking requests.
func WithOpenVikingTenant(ctx context.Context, accountID, userID string) context.Context {
	ctx = context.WithValue(ctx, openVikingAccountContextKey, strings.TrimSpace(accountID))
	return context.WithValue(ctx, openVikingUserContextKey, strings.TrimSpace(userID))
}

func openVikingTenantFromContext(ctx context.Context) (string, string, bool) {
	if ctx == nil {
		return "", "", false
	}
	accountID, _ := ctx.Value(openVikingAccountContextKey).(string)
	userID, _ := ctx.Value(openVikingUserContextKey).(string)
	accountID = strings.TrimSpace(accountID)
	userID = strings.TrimSpace(userID)
	return accountID, userID, accountID != "" && userID != ""
}
