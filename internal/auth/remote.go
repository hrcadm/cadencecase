package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/yourname/sleeptracker/internal"
)

type RemoteAuthProvider struct {
	AuthServiceURL string
	HTTPClient     *http.Client
	logger         internal.Logger
}

func (a *RemoteAuthProvider) ValidateTokenLocal(token string) (*internal.User, error) {
	return nil, errors.New("not implemented in RemoteAuthProvider")
}

func (a *RemoteAuthProvider) ValidateTokenRemote(ctx context.Context, token string) (*internal.User, error) {
	body := `{"token":"` + token + `"}`
	req, err := http.NewRequestWithContext(ctx, "POST", a.AuthServiceURL, strings.NewReader(body))
	if err != nil {
		a.logger.Errorf("failed to create request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.logger.Errorf("failed to call auth service: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		a.logger.Errorf("auth service returned %d", resp.StatusCode)
		return nil, errors.New("auth service returned non-200")
	}
	var user internal.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		a.logger.Errorf("failed to decode auth response: %v", err)
		return nil, err
	}
	return &user, nil
}

func NewRemoteAuthProvider(url string, logger internal.Logger) *RemoteAuthProvider {
	return &RemoteAuthProvider{
		AuthServiceURL: url,
		HTTPClient:     &http.Client{Timeout: 5 * time.Second},
		logger:         logger,
	}
}
