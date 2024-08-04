package app_tests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boo-admin/boo"
	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/engine/echofunctions"
	"github.com/boo-admin/boo/engine/echosrv"
	"github.com/boo-admin/boo/goutils/httpext"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/services/authn/base_auth"
	"github.com/boo-admin/boo/services/authn/jwt_auth"
	"github.com/boo-admin/boo/services/authn/session_auth"
	"github.com/golang-jwt/jwt/v4"
	_ "github.com/lib/pq"
	gobatis "github.com/runner-mei/GoBatis"
	"golang.org/x/exp/slog"
)

type TestApp struct {
	Logger     *slog.Logger
	Env        *client.Environment
	CurrentDir string
	Server     *boo.Server

	runner *httpext.Runner
}

func (a *TestApp) DbNow() time.Time {
	sqlstr := "select now()"
	if dialect := a.Server.Factory.Dialect(); dialect == gobatis.DM || dialect == gobatis.Oracle {
		sqlstr = "select now() from dual"
	}
	var dbnow time.Time
	err := a.Server.Factory.DB().QueryRowContext(context.Background(), sqlstr).Scan(&dbnow)
	if err != nil {
		panic(err)
	}

	return dbnow
}

func setDefault(params map[string]string, key, value string) {
	if _, ok := params[key]; !ok {
		params[key] = value
	}
}

func NewTestApp(t testing.TB, params map[string]string) *TestApp {
	currentDir, err := GetPackageRoot()
	if err != nil {
		t.Error("GetPackageRoot()", err)
		t.FailNow()
	}
	if params == nil {
		params = map[string]string{}
	}
	setDefault(params, "log.filename", "console")
	setDefault(params, "db.reset_db", "true")
	setDefault(params, "db.drv", "postgres")
	setDefault(params, "db.url", "host=127.0.0.1 port=5432 user=golang password=123456 dbname=golang sslmode=disable")

	env, err := client.NewEnvironmentWith("boo", "test.properties", params)
	if err != nil {
		t.Error("NewEnvironmentWith()", err)
		t.FailNow()
	}
	return &TestApp{
		Logger:     env.Logger,
		Env:        env,
		CurrentDir: currentDir,
	}
}

func (app *TestApp) BaseURL() string {
	return app.runner.MustURL() + "/boo/api/v1"
}

func (app *TestApp) Start(t testing.TB) {
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
	jwtAuth, err := jwt_auth.New(app.Env, jwtUser)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	sessionUser := func(ctx context.Context, req *http.Request, values url.Values) (context.Context, error) {
		username := values.Get(session_auth.SESSION_USER_KEY)
		return authn.ContextWithReadCurrentUser(ctx, authn.ReadCurrentUserFunc(func(ctx context.Context) (authn.AuthUser, error) {
			return authn.NewMockUser(username), nil
		})), nil
	}
	sessionAuth, err := session_auth.New(app.Env, sessionUser)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	validator := func(ctx context.Context, req *http.Request, username string, password string) (context.Context, error) {
		if username == "admin" && password == "admin" {
			return authn.ContextWithReadCurrentUser(ctx, authn.ReadCurrentUserFunc(func(stdctx context.Context) (authn.AuthUser, error) {
				return authn.NewMockUser("admin"), nil
			})), nil
		}
		return ctx, authn.ErrInvalidCredentials
	}
	baseAuth /* , err */ := base_auth.Verify(validator)
	// if err != nil {
	//	t.Error(err)
	//	t.FailNow()
	// }

	var validateFns = []authn.AuthValidateFunc{
		jwtAuth,
		sessionAuth,
		baseAuth,
	}
	echosrv.Use(echofunctions.HTTPAuth(nil, validateFns...))

	srv, err := boo.NewServer(app.Env)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	app.Server = srv

	engine, err := echosrv.New(srv, "/boo/api/v1")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	runner := httpext.NewRunner(app.Logger, ":1323")
	err = runner.Start(context.Background(), engine)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	app.runner = runner
}

func (app *TestApp) Stop(t testing.TB) {
	if app.runner == nil {
		return
	}
	err := app.runner.Stop(context.Background())
	if err != nil {
		t.Error(err)
	}
}

// AbsoluteToImport gets an absolute path and tryes to transform it into
// a valid package import path. E.g., if $GOPATH is "/home/user/go" then the path:
//
//	/home/user/go/src/github.com/colegion/goal
//
// must be transformed into:
//
//	github.com/colegion/goal
//
// The path must be within "$GOPATH/src", otherwise an error will be returned.
func AbsoluteToImport(abs string) (string, error) {
	// Make sure the input path is in fact absolute.
	if !filepath.IsAbs(abs) {
		return "", fmt.Errorf(`absolute path expected, got "%s"`, abs)
	}

	// Check every $GOPATH whether some of them is a prefix of the input path.
	// That would mean the input path is within $GOPATH.
	gopaths := filepath.SplitList(os.Getenv("GOPATH"))
	for i := 0; i < len(gopaths); i++ {
		// Getting a "$GOPATH/src".
		gopath := filepath.Join(gopaths[i], "src")

		// Checking whether "$GOPATH/src" is a prefix of the input path.
		if res := strings.TrimPrefix(abs, gopath); res != abs {
			// Return the "$GOPATH/src"-less version of the path.
			// Make sure "/" are used as separators and there are no
			// leading or trailing slashes.
			return strings.Trim(filepath.ToSlash(res), "/"), nil
		}
	}

	// If no import path returned so far, requested path is not inside "$GOPATH/src".
	return "", fmt.Errorf(`path "%s" is not inside "$GOPATH/src"`, abs)
}

// ImportToAbsolute gets a valid package import path and tries to transform
// it into an absolute path. E.g., there is an input:
//
//	github.com/username/project
//
// It will output:
//
//	$GOPATH/src/github.com/username/project
//
// NOTE: The first value from the list of GOPATHs is always used.
func ImportToAbsolute(imp string) (string, error) {
	// Make sure the input import path is not relative.
	var err error
	imp, err = CleanImport(imp)
	if err != nil {
		return "", err
	}

	// Replace the "/" by the platform specific separators.
	p := filepath.FromSlash(imp)

	// Make sure the path is not a valid absolute path.
	if filepath.IsAbs(p) {
		return p, nil
	}

	// Split $GOPATH list to use the first value.
	gopaths := filepath.SplitList(os.Getenv("GOPATH"))

	for _, gopa := range gopaths {
		absPath := filepath.Join(gopa, "src", p)
		if st, err := os.Stat(absPath); err == nil && st.IsDir() {
			return absPath, nil
		}
	}

	// Join input path with the "$GOPATH/src" and return.
	// Make sure $GOPATH is normalized (i.e. unix style delimiters are used).
	return "", os.ErrNotExist
}

// CleanImport gets a package import path and returns it as is if it is absolute.
// Otherwise, it tryes to convert it to an absolute form.
func CleanImport(imp string) (string, error) {
	// If the path is not relative, return it as is.
	impNorm := filepath.ToSlash(imp)
	if impNorm != "." && impNorm != ".." &&
		!filepath.HasPrefix(impNorm, "./") &&
		!filepath.HasPrefix(impNorm, "../") {

		// Get rid of trailing slashes.
		return strings.TrimRight(impNorm, "/"), nil
	}

	// Find a full absolute path to the requested import.
	abs, err := filepath.Abs(filepath.FromSlash(imp))
	if err != nil {
		return "", err
	}

	// Extract package's import from it.
	return AbsoluteToImport(abs)
}

func GetModulePath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		gomod := filepath.Join(wd, "go.mod")
		if client.FileExists(gomod) {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if len(parent) >= len(wd) {
			return "", errors.New("没有找到 go.mod")
		}
		wd = parent
	}
}

func GetPackageRoot() (string, error) {
	rootDir, err := ImportToAbsolute("github.com/boo-admin/boo")
	if err == nil {
		return rootDir, nil
	}

	return GetModulePath()
}