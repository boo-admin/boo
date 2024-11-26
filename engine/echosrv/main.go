package echosrv

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/boo-admin/boo"
	"github.com/boo-admin/boo/booclient"
	"github.com/boo-admin/boo/engine/echofunctions"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/httpext"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/services/authn/base_auth"
	"github.com/boo-admin/boo/services/authn/jwt_auth"
	"github.com/boo-admin/boo/services/authn/session_auth"
	"github.com/boo-admin/boo/services/docs"
	"github.com/boo-admin/boo/services/users"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger" // echo-swagger middleware
	_ "github.com/swaggo/files/v2"               // swagger embed files
)

var middlewares []echo.MiddlewareFunc

func Use(middleware ...echo.MiddlewareFunc) {
	middlewares = append(middlewares, middleware...)
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
	booclient.InitOperationQueryer(mux, srv.OperationQueryer)
	booclient.InitDepartments(mux, srv.Departments)
	booclient.InitUsers(mux, srv.Users)
	booclient.InitUserTags(mux, srv.UserTags)
	users.InitUsersForHTTP(mux, srv.Users)
	booclient.InitRoles(mux, srv.Roles)
	booclient.InitEmployees(mux, srv.Employees)
	users.InitEmployeesForHTTP(mux, srv.Employees)
	booclient.InitEmployeeTags(mux, srv.EmployeeTags)

	return e, nil
}

func Run(srv *boo.Server, prefix, listenAt string) error {
	engine, err := New(srv, prefix)
	if err != nil {
		return err
	}

	jwtUser := func(ctx context.Context, req *http.Request, token *jwt.Token) (context.Context, error) {
		claims, ok := token.Claims.(*jwt.StandardClaims)
		if !ok {
			return nil, errors.New("claims not jwt.StandardClaims")
		}

		ss := strings.SplitN(claims.Audience, " ", 2)
		if len(ss) < 2 {
			return nil, errors.New("Audience '" + claims.Audience + "' is invalid")
		}
		// userid := ss[0]
		username := ss[1]

		return authn.ContextWithReadCurrentUser(ctx, authn.ReadCurrentUserFunc(func(ctx context.Context) (authn.AuthUser, error) {
			return authn.NewMockUser(username), nil
		})), nil
	}
	jwtAuth, err := jwt_auth.New(srv.Env, jwtUser)
	if err != nil {
		return errors.Wrap(err, "init jwt auth")
	}

	sessionUser := func(ctx context.Context, req *http.Request, values url.Values) (context.Context, error) {
		username := values.Get(session_auth.SESSION_USER_KEY)
		return authn.ContextWithReadCurrentUser(ctx, authn.ReadCurrentUserFunc(func(ctx context.Context) (authn.AuthUser, error) {
			return authn.NewMockUser(username), nil
		})), nil
	}
	sessionAuth, err := session_auth.New(srv.Env, sessionUser)
	if err != nil {
		return errors.Wrap(err, "init session auth")
	}

	validator := func(ctx context.Context, req *http.Request, username string, password string) (context.Context, error) {
		if username == "admin" && password == "admin" {
			user := authn.NewMockUser("admin")
			return authn.ContextWithReadCurrentUser(ctx,
				authn.ReadCurrentUserFunc(func(stdctx context.Context) (authn.AuthUser, error) {
					return user, nil
				})), nil
		}
		return ctx, authn.ErrInvalidCredentials
	}
	baseAuth /* , err */ := base_auth.Verify(validator)
	// if err != nil {
	// 	return errors.Wrap(err, "init base auth")
	// }

	var validateFns = []authn.AuthValidateFunc{
		jwtAuth,
		sessionAuth,
		baseAuth,
	}
	Use(echofunctions.HTTPAuth(nil, validateFns...))

	runner := httpext.NewRunner(srv.Env.Logger, listenAt)
	return runner.Run(context.Background(), engine)
}
