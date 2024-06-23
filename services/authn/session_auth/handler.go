package session_auth

import (
	"context"
	"crypto/sha1"
	"hash"
	"net/http"
	"net/url"
	"strings"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/services/authn"
)

type Option struct {
	SessionPath string
	SessionKey  string
	SessionHash func() hash.Hash
	SecretKey   []byte
	// CurrentURL  func(*http.Request) url.URL
}

func SessionVerify(opt *Option, handle func(ctx context.Context, req *http.Request, values url.Values) (context.Context, error)) authn.AuthValidateFunc {
	sessionKey := opt.SessionKey
	if sessionKey == "" {
		sessionKey = DefaultSessionKey
	}
	sessionPath := opt.SessionPath
	secretKey := opt.SecretKey

	if sessionPath == "" {
		sessionPath = "/" // 必须指定 Path, 否则会被自动赋成当前请求的 url 中的 path
	} else if !strings.HasPrefix(sessionPath, "/") {
		sessionPath = "/" + sessionPath
	}

	// currentURL := opt.CurrentURL
	// if currentURL == nil {
	// 	currentURL = func(req *http.Request) url.URL {
	// 		return *req.URL
	// 	}
	// }

	h := opt.SessionHash
	if h == nil {
		h = sha1.New
	}

	return func(ctx context.Context, req *http.Request) (context.Context, error) {
		values, err := GetValues(req, sessionKey, h, secretKey)
		if err != nil {
			if err == ErrCookieNotFound || err == ErrCookieEmpty {
				return nil, authn.ErrTokenNotFound
			}
			return nil, err
		}

		return handle(ctx, req, values)
	}
}

func New(env *client.Environment, sessionUser func(ctx context.Context, req *http.Request, values url.Values) (context.Context, error)) (authn.AuthValidateFunc, error) {
	var sessionOpt Option
	sessionOpt.SessionPath = env.Config.StringWithDefault("auth.session.cookie_path", "/")
	sessionOpt.SessionKey = env.Config.StringWithDefault("auth.session.cookie_name", "boo_session")

	var sessionHash = env.Config.StringWithDefault("auth.session.hash_method", "sha1")
	if h, err := GetHash(sessionHash); err != nil {
		return nil, errors.Wrap(err, "参数 'session-hash-method' 的值 '"+sessionHash+"' 是未知的 hash 算法")
	} else {
		sessionOpt.SessionHash = h
	}
	sessionOpt.SecretKey = []byte(env.Config.StringWithDefault("auth.session.secret_key", ""))

	return SessionVerify(&sessionOpt, sessionUser), nil
}
