package ctxdata

import "context"

func GetUid(ctx context.Context) string {
	if u, ok := ctx.Value(Identify).(string); ok {
		return u
	}
	return ""
}
