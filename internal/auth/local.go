package auth

import (
	"context"
	"errors"

	"github.com/yourname/sleeptracker/internal"
)

type LocalAuthProvider struct {
	Token  string
	logger internal.Logger
}

func (a *LocalAuthProvider) ValidateTokenLocal(token string) (*internal.User, error) {
	if token == a.Token {
		return &internal.User{ID: "u1", Token: a.Token, Name: "Demo User"}, nil
	}
	a.logger.Warnf("invalid token: %s", token)
	return nil, errors.New("invalid token")
}

func (a *LocalAuthProvider) ValidateTokenRemote(ctx context.Context, token string) (*internal.User, error) {
	a.logger.Warnf("ValidateTokenRemote not implemented in LocalAuthProvider")
	return nil, errors.New("not implemented in LocalAuthProvider")
}

func NewLocalAuthProvider(token string, logger internal.Logger) *LocalAuthProvider {
	return &LocalAuthProvider{Token: token, logger: logger}
}
