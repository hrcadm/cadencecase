package auth

import (
	"context"

	"github.com/yourname/sleeptracker/internal"
)

type Provider interface {
	ValidateTokenLocal(token string) (*internal.User, error)
	ValidateTokenRemote(ctx context.Context, token string) (*internal.User, error)
}
