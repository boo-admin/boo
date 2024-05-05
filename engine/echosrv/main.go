package echosrv

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/boo-admin/boo"
	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/engine/echofunctions"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/services/docs"
	"github.com/boo-admin/boo/services/users"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger" // echo-swagger middleware
	_ "github.com/swaggo/files"                  // swagger embed files
)

func Start(srv *boo.Server) (io.Closer, error) {
	errC := make(chan error, 1)

	var closer boo.SyncCloser
	go func() {
		err := run(srv, &closer)
		if err != nil {
			errC <- err
		}
	}()

	timer := time.NewTimer(5 * time.Second)
	select {
	case err := <-errC:
		return nil, err
	case <-timer.C:
		return &closer, nil
	}
}

func Run(srv *boo.Server) error {
	var closer boo.SyncCloser
	return run(srv, &closer)
}

func run(srv *boo.Server, closer *boo.SyncCloser) error {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Validator: middleware.BasicAuthValidator(func(username string, password string, ctx echo.Context) (bool, error) {
			if username == "admin" && password == "admin" {
				c := echofunctions.GetContext(ctx)
				user := authn.NewMockUser("admin")
				c = authn.ContextWithReadCurrentUser(c,
					authn.ReadCurrentUserFunc(func(stdctx context.Context) (authn.AuthUser, error) {
						return user, nil
					}))
				echofunctions.SetContext(ctx, c)
				return true, nil
			}
			return false, nil
		}),
	}))

	// Routes
	mux := e.Group("/boo/api/v1")
	EnalbeSwaggerAt(e, "/boo/swagger", docs.SwaggerInfo.InstanceName())
	client.InitOperationQueryer(mux, srv.OperationQueryer)
	client.InitDepartments(mux, srv.Departments)
	client.InitUsers(mux, srv.Users)
	users.InitUsersForHTTP(mux, srv.Users)
	client.InitEmployees(mux, srv.Employees)
	users.InitEmployeesForHTTP(mux, srv.Employees)

	closer.Set(client.CloseFunc(func() error {
		return e.Shutdown(context.Background())
	}))
	defer closer.Set(nil)

	err := e.Start(":1323")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func EnalbeSwaggerAt(e *echo.Echo, prefix, instanceName string) {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if strings.HasSuffix(prefix, "/") {
		prefix = strings.TrimSuffix(prefix, "/")
	}

	handler := echoSwagger.EchoWrapHandler(echoSwagger.InstanceName(instanceName))

	mux := e.Group(prefix, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rawRequestURI := c.Request().RequestURI
			index := strings.Index(rawRequestURI, prefix)
			if index >= 0 {
				c.Request().RequestURI = rawRequestURI[index:]
			}
			if c.Request().RequestURI == prefix ||
				c.Request().RequestURI == prefix+"/" {
				if strings.HasSuffix(rawRequestURI, "/") {
					return c.Redirect(http.StatusTemporaryRedirect, rawRequestURI+"index.html")
				} else {
					return c.Redirect(http.StatusTemporaryRedirect, rawRequestURI+"/index.html")
				}
			}
			return next(c)
		}
	})
	mux.Any("/*", handler)
}
