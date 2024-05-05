package echofunctions

import (
	"context"

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
