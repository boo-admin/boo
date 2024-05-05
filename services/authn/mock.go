package authn

import (
	"context"
)

func NewMockUser(name string) AuthUser {
	return &mockUser{name: name}
}

// AuthUser 用户信息
type mockUser struct {
	name string
}

func (*mockUser) ID() int64 {
	return 0
}

func (m *mockUser) Name() string {
	return m.name
}

func (m *mockUser) Nickname() string {
	return m.name
}

func (m *mockUser) DisplayName(ctx context.Context, fmt ...string) string {
	return m.name
}

func (m *mockUser) WriteProfile(key, value string) error {
	return nil
}

func (m *mockUser) ReadProfile(key string) (string, error) {
	return "", nil
}

func (m *mockUser) Data(ctx context.Context, key string) interface{} {
	return nil
}

func (m *mockUser) RoleIDs() []int64 {
	return nil
}

func (m *mockUser) RoleNames() []string {
	return nil
}

func (m *mockUser) HasPermission(ctx context.Context, permissionID string) (bool, error) {
	return true, nil
}

func (m *mockUser) HasPermissionAny(ctx context.Context, permissionIDs []string) (bool, error) {
	return true, nil
}

func (m *mockUser) HasRole(string) bool {
	return true
}

func (m *mockUser) HasRoleID(id int64) bool {
	return true
}

func (m *mockUser) ForEach(func(string, interface{})) {
}
