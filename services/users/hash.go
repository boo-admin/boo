package users

import (
	"context"
	"encoding/hex"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/errors"
	"golang.org/x/crypto/bcrypt"
)

var passwordHashers = map[string]func(env *client.Environment) (UserPassworder, error){}

func RegisterPassworder(alg string, factory func(env *client.Environment) (UserPassworder, error)) {
	passwordHashers[alg] = factory
}

func NewUserPassworder(env *client.Environment) (UserPassworder, error) {
	alg := env.Config.StringWithDefault("users.password_hash_alg", "")
	if alg != "" && alg != "default" {
		f, ok := passwordHashers[alg]
		if !ok {
			return nil, errors.New("用户密码加密算法 '" + alg + "' 不支持")
		}
		return f(env)
	}

	return &userPasswordHasher{
		cost: bcrypt.DefaultCost,
	}, nil
}

type userPasswordHasher struct {
	cost int
}

func (h *userPasswordHasher) Hash(ctx context.Context, password string) (string, error) {
	bs, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bs), nil
}

func (h *userPasswordHasher) Compare(ctx context.Context, password, hashedPassword string) error {
	hashedPwdBytes, err := hex.DecodeString(hashedPassword)
	if err != nil {
		return err
	}
	return bcrypt.CompareHashAndPassword(hashedPwdBytes, []byte(password))
}
