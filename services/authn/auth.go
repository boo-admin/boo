package authn

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/boo-admin/boo/errors"
	"golang.org/x/exp/slog"
)

func NewHTTPError(code int, msg string) error {
	return errors.WithCode(errors.New(msg), code)
}

var (
	ErrUnauthorized       = NewHTTPError(http.StatusUnauthorized, "auth: token is unauthorized")
	ErrTokenExpired       = NewHTTPError(http.StatusUnauthorized, "auth: token is expired")
	ErrTokenNotFound      = NewHTTPError(http.StatusUnauthorized, "auth: no token found")
	ErrUserNotFound       = NewHTTPError(http.StatusForbidden, "auth: user isnot exists")
	ErrInvalidCredentials = NewHTTPError(http.StatusForbidden, "auth: invalid credentials")

	// 仅用于 token 找到，但不适用检验函数的时候
	ErrSkipped = errors.ErrSkipped
)

func ReturnError(ctx context.Context, w http.ResponseWriter, r *http.Request, code int, err error) {
	encodedError := errors.ToEncodeError(err, code)

	RenderJSON(ctx, w, r, encodedError.HTTPCode(), encodedError)
}

func RenderJSON(ctx context.Context, w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		slog.WarnContext(ctx, "encode http response fail",
			slog.String("method", r.Method),
			slog.String("url", r.URL.Path),
			slog.Any("error", err))
	}
}

type AuthValidateFunc func(ctx context.Context, req *http.Request) (context.Context, error)
type ContextHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func RawHTTPAuth(returnError func(context.Context, http.ResponseWriter, *http.Request, string, int), validateFns ...AuthValidateFunc) func(ContextHandlerFunc) ContextHandlerFunc {
	if returnError == nil {
		returnError = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err string, statusCode int) {
			RenderJSON(ctx, w, r, statusCode, map[string]interface{}{
				"code":  statusCode,
				"error": err,
			})
		}
	}

	return func(next ContextHandlerFunc) ContextHandlerFunc {
		hfn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			for _, fn := range validateFns {
				nctx, err := fn(ctx, r)
				if err == nil {
					next(nctx, w, r)
					return
				}

				if err != ErrTokenNotFound {
					returnError(ctx, w, r, err.Error(), http.StatusUnauthorized)
					return
				}
			}
			returnError(ctx, w, r, ErrTokenNotFound.Error(), http.StatusUnauthorized)
		}
		return ContextHandlerFunc(hfn)
	}
}
