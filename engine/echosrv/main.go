package echosrv

import (
	"context"
	"net/http"
	"strings"

	"github.com/boo-admin/boo"
	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/engine/echofunctions"
	"github.com/boo-admin/boo/goutils/httpext"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/services/docs"
	"github.com/boo-admin/boo/services/users"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger" // echo-swagger middleware
	_ "github.com/swaggo/files/v2"               // swagger embed files
)

var middlewares []echo.MiddlewareFunc

func Use(middleware ...echo.MiddlewareFunc) {
	middlewares = append(middlewares, middleware...)
}

func TestAuth() echo.MiddlewareFunc {
	return middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
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
	})
}

func EnalbeSwaggerAt(e *echo.Group, prefix, instanceName string) {
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

func New(srv *boo.Server, prefix string) (*echo.Echo, error) {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	for _, m := range middlewares {
		e.Use(m)
	}

	// Routes
	mux := e.Group(prefix)
	EnalbeSwaggerAt(mux, "/swagger", docs.SwaggerInfobooswagger.InstanceName())
	client.InitOperationQueryer(mux, srv.OperationQueryer)
	client.InitDepartments(mux, srv.Departments)
	client.InitUsers(mux, srv.Users)
	users.InitUsersForHTTP(mux, srv.Users)
	client.InitEmployees(mux, srv.Employees)
	users.InitEmployeesForHTTP(mux, srv.Employees)

	return e, nil
}

func Run(srv *boo.Server, prefix, listenAt string) error {
	engine, err := New(srv, prefix)
	if err != nil {
		return err
	}

	runner := httpext.NewRunner(srv.Logger, listenAt)
	return runner.Run(context.Background(), engine)
}
