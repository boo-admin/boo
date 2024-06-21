package echofunctions

import (
	"context"
	"net/http"

	"github.com/boo-admin/boo/services/authn"
	"github.com/labstack/echo/v4"
)

func GetContext(ctx echo.Context) context.Context {
	o := ctx.Get("stdcontext")
	if o == nil {
		return ctx.Request().Context()
	}
	c, ok := o.(context.Context)
	if !ok {
		return ctx.Request().Context()
	}
	return c
}

func SetContext(ctx echo.Context, stdctx context.Context) {
	ctx.Set("stdcontext", stdctx)
}

func HTTPAuth(returnError func(echo.Context, string, int) error, validateFns ...authn.AuthValidateFunc) echo.MiddlewareFunc {
	if returnError == nil {
		returnError = func(ctx echo.Context, err string, statusCode int) error {
			return ctx.JSON(statusCode, map[string]interface{}{
				"code":  statusCode,
				"error": err,
			})
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			nctx := GetContext(ctx)
			for _, fn := range validateFns {
				nctx, err := fn(nctx, ctx.Request())
				if err == nil {
					SetContext(ctx, nctx)
					return next(ctx)
				}

				if err != authn.ErrTokenNotFound {
					return returnError(ctx, err.Error(), http.StatusUnauthorized)
				}
			}
			return returnError(ctx, authn.ErrTokenNotFound.Error(), http.StatusUnauthorized)
		}
	}
}
