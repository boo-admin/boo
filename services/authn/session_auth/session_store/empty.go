package session_store

import (
	"context"

	"github.com/boo-admin/boo/booclient"
	"github.com/boo-admin/boo/services/authn/session_auth"
)

func CreateEmpty(env *booclient.Environment) session_auth.Onlines {
	return &EmptySessions{}
}

type EmptySessions struct{}

func (sess EmptySessions) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (sess EmptySessions) Login(ctx context.Context, username, address, apiKey string) (string, error) {
	return "", nil
}
func (sess EmptySessions) LogoutBySessionID(ctx context.Context, key string) error {
	return nil
}
func (sess EmptySessions) LogoutByUsername(ctx context.Context, username string) error {
	return nil
}
func (sess EmptySessions) GetBySessionID(ctx context.Context, id string) (*booclient.OnlineInfo, error) {
	return nil, nil
}
func (sess EmptySessions) Query(ctx context.Context, username string) ([]booclient.OnlineInfo, error) {
	return nil, nil
}
func (sess EmptySessions) List(ctx context.Context) ([]booclient.OnlineInfo, error) {
	return nil, nil
}
func (sess EmptySessions) UpdateNow(ctx context.Context, uuid, apiKey string) error {
	return nil
}
func (sess EmptySessions) DeleteExpired(ctx context.Context) error {
	return nil
}
func (sess EmptySessions) IsOnlineExists(ctx context.Context, username, loginAddress string) error {
	return nil
}
