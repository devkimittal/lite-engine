package client

import "context"

type ContextKey string

const (
	Hosted = ContextKey("hosted")
)

func IsHosted(ctx context.Context) bool {
	isHosted := ctx.Value(Hosted)
	if isHosted == nil {
		return false
	}
	return isHosted.(bool)
}
