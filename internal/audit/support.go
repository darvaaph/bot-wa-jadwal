package audit

import "context"

type supportKey struct{}

func WithSupport(ctx context.Context, grantID, classID int64, reason string) context.Context {
	return context.WithValue(ctx, supportKey{}, map[string]any{"support_grant_id": grantID, "support_class_id": classID, "support_reason": reason})
}
