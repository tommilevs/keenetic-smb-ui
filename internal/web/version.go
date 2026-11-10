package web

import (
	"context"
	"time"
)

// Version is injected at build time via -ldflags "-X ...web.Version=...".
var Version = "dev"

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
