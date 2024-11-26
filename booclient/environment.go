package booclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/boo-admin/boo/goutils/as"
	"github.com/mei-rune/properties"
	"golang.org/x/exp/slog"
	"gopkg.in/natefinch/lumberjack.v2"
)

const CfgRunMode = "runMode"
const DevRunMode = "dev"
const TestRunMode = "test"

// 本系统有三种部署方式
// 1. 单体程序
//
//	+-------------+
//	|     app1    |
//	+-------------+
//
// 2. 多 app + gateway 方式
//
// +-------------------------------------------------+
// |                host 1                           |
// |                                                 |
// |           +------------------+                  |
// |           |      gateway     |------------------+
// |           +---------+--------+                  |
// |                     |                           |
// |       +-------------+------------------+        |
// |       |             |                  |        |
// | +-----+-------+ +-------------+ +------+------+ |
// | |     app1    | |    app2     | |     app3    | |
// | +-------------+ +-------------+ +------------ + |
// |                                                 |
// +-------------------------------------------------+
//
// 3. 多主机(多app) + gateway 方式
//
// +-------------------------------------------------+   +-----------------------------------------------+
// |                host 1                           |   |               host 2                          |
// |                                                 |   |                                               |
// |              +------------------+               |   |                                               |
// |              |      gateway     |---------------+---+------------+                                  |
// |              +---------+--------+               |   |            |                                  |
// |                        |                        |   |            |                                  |
// |       +----------------+---------------+        |   |            |                                  |
// |       |                |               |        |   |            |                                  |
// | +-----+-------+ +-------------+ +------+------+ |   |    +-------+---------+                        |
// | |     app1    | |    app2     | |     app3    | |   |    |      app3       |                        |
// | +-------------+ +-------------+ +------------ + |   |    +-----------------+                        |
// |                                                 |   |                                               |
// +-------------------------------------------------+   +-----------------------------------------------+
//
// 这三种部署方式中后二种内部各个 app 需要相互 rpc 调用对方提供的服务， 如果要调用 rpc 服务的话就需要获得对方的地址
// 1.  系统内部用户,  统一通过 gateway 访问，
//
//	系统外部用户
//
// 2.
// 用户访问本系统时的 地址
const CfgSystemHomeEndpoint = "system.app_home"

// 主地址
const CfgSystemMasterApiEndpoint = "system.api.master"

// 本地地址
const CfgSystemLocalApiEndpoint = "system.api.local"

var DefaultHeaderTitleText = "head test"
var DefaultFooterTitleText = "foot test"
var DefaultProductName = "boo"
var DefaultURLPath = "boo"
var Version = "v1"
var FullVersion = "v1.0.0"
var DefultConfigFilename = "boo.log"
var isWindows = runtime.GOOS == "windows"

type Environment struct {
	Logger               *slog.Logger
	HeaderTitleText      string
	FooterTitleText      string
	LoginHeaderTitleText string
	LoginFooterTitleText string

	Namespace   string
	Name        string
	Version     string
	FullVersion string
	Config      *Config
	Fs          FileSystem
	RunMode     string

	// 当前应用的路径（后面有斜杠），不包括： 协议，IP ＋　端口
	AppPathWithSlash string
	// 当前应用的路径（后面没有斜杠），不包括： 协议，IP ＋　端口
	AppPathWithoutSlash string

	// AppPathWithSlash 的别名， 为了兼容老程序
	DaemonUrlPath string
}

func (env *Environment) IsTestMode() bool {
	return env.RunMode == TestRunMode
}

func (env *Environment) IsMasterNode() bool {
	isEnabled := env.Config.BoolWithDefault("engine.is_enabled", false)
	if !isEnabled {
		return true
	}
	name := strings.TrimSpace(env.Config.StringWithDefault("engine.name", "default"))
	return strings.ToLower(name) == "default"
}

// 访问 url, 注意它包括了协议，地址，端口，不包含根路径
func (env *Environment) GetApiEndpoint() (s string) {
	if env.IsMasterNode() {
		s = env.Config.StringWithDefault(CfgSystemMasterApiEndpoint, "")
	} else {
		s = env.Config.StringWithDefault(CfgSystemLocalApiEndpoint, "")
	}
	if s == "" {
		panic("CfgSystemMasterApiEndpoint missing")
	}
	return s
}

func (env *Environment) GetHomeEndpoint() (s string) {
	s = env.Config.StringWithDefault(CfgSystemHomeEndpoint, "")
	if s == "" {
		panic("GetHomeEndpoint missing")
	}
	return s
}

// 访问 url, 注意它包括了协议，地址，端口，以及根路径
func (env *Environment) GetApiURL(opt ...URLOption) string {
	var o URLOption = WithHostPort
	if len(opt) > 0 {
		o = opt[0]
	}

	switch o.value() {
	case WithHostPort:
		u := env.GetApiEndpoint()
		return Urljoin(u, env.AppPathWithoutSlash)
	case WithoutHostPort:
		return env.AppPathWithoutSlash
	default:
		panic(fmt.Errorf("unknown url option - %T", o))
	}
}

func ReadFileWithDefault(logger *slog.Logger, files []string, defaultValue string) string {
	for _, s := range files {
		content, e := ioutil.ReadFile(s)
		if e == nil {
			return string(bytes.TrimSpace(content))
		}
		if !os.IsNotExist(e) {
			logger.Warn("ReadFileWithDefault fail", slog.Any("error", e))
		}
	}
	return defaultValue
}

func NewEnvironmentWith(namespace, filename string, defaultValues map[string]string) (*Environment, error) {
	fs, err := NewFileSystem(namespace, map[string]string{})
	if err != nil {
		return nil, err
	}

	var nonexistfilenames, existfilenames []string

	var defaultFiles []string
	var customFiles []string
	defaultFiles = append(defaultFiles, fs.FromConfig(DefultConfigFilename))
	customFiles = append(customFiles, fs.FromCustomConfig(DefultConfigFilename))
	customFiles = append(customFiles, filename)

	var allProps = map[string]string{}
	for key, value := range defaultValues {
		allProps[key] = value
	}
	for _, names := range [][]string{
		defaultFiles,
		customFiles,
	} {
		for _, name := range names {
			props, err := properties.ReadProperties(name)
			if err != nil {
				if !os.IsNotExist(err) {
					return nil, err
				}
				nonexistfilenames = append(nonexistfilenames, name)
				continue
			}
			existfilenames = append(existfilenames, name)
			for key, value := range props {
				allProps[key] = value
			}
		}
	}
	initFsWithConfig(fs, namespace, allProps)
	cfg := NewConfigWith(allProps)

	logger := NewLogger(namespace, cfg, fs, nil)
	if len(allProps) > 0 {
		if value := allProps["test_node_name"]; value != "" {
			logger = logger.With(slog.String("node_name", value))
		}
	}
	if logger.Enabled(nil, slog.LevelDebug) {
		logger.Debug("load config successful",
			slog.Any("existnames", existfilenames),
			slog.Any("nonexistnames", nonexistfilenames))
	}

	env := NewEnvironment(namespace, cfg, fs, logger)
	return env, nil
}

func NewEnvironment(namespace string, cfg *Config, fs FileSystem, logger *slog.Logger) *Environment {
	appURL := cfg.StringWithDefault("app.urlpath", DefaultURLPath)
	if !strings.HasPrefix(appURL, "/") {
		appURL = "/" + appURL
	}
	if strings.HasSuffix(appURL, "/") {
		appURL = strings.TrimSuffix(appURL, "/")
	}

	env := &Environment{
		Logger:              logger,
		Namespace:           namespace,
		Name:                cfg.StringWithDefault("product.name", DefaultProductName),
		Config:              cfg,
		Fs:                  fs,
		Version:             Version,
		FullVersion:         FullVersion,
		AppPathWithoutSlash: appURL,
		AppPathWithSlash:    appURL + "/",
		DaemonUrlPath:       appURL + "/",
		RunMode:             cfg.StringWithDefault(namespace+"."+CfgRunMode, ""),
	}
	if env.RunMode == "" {
		env.RunMode = os.Getenv(namespace + "_run_mode")
	}
	// env.FullVersion = ReadFileWithDefault(logger,
	// 		[]string{ fs.FromInstallRoot("VERSION") },
	// 		Version)
	// if ss := strings.Split(env.FullVersion, "."); len(ss) >= 2 {
	// 	env.Version = ss[0] + "." + ss[1]
	// } else {
	// 	env.Version = env.FullVersion
	// }

	env.HeaderTitleText = cfg.StringWithDefault("product.header_title",
		ReadFileWithDefault(logger, []string{
			fs.FromCustomConfig("resources/profiles/header.txt"),
			fs.FromData("resources/profiles/header.txt")},
			DefaultHeaderTitleText))

	env.FooterTitleText = cfg.StringWithDefault("product.footer_title",
		ReadFileWithDefault(logger, []string{
			fs.FromCustomConfig("resources/profiles/footer.txt"),
			fs.FromData("resources/profiles/footer.txt")},
			DefaultFooterTitleText))

	env.LoginHeaderTitleText = cfg.StringWithDefault("product.login_header_title",
		ReadFileWithDefault(logger, []string{
			fs.FromCustomConfig("resources/profiles/login-title.txt"),
			fs.FromData("resources/profiles/login-title.txt")},
			env.HeaderTitleText))

	env.LoginFooterTitleText = cfg.StringWithDefault("product.login_footer_title",
		ReadFileWithDefault(logger, []string{
			fs.FromCustomConfig("resources/profiles/login-footer.txt"),
			fs.FromData("resources/profiles/login-footer.txt")},
			env.FooterTitleText))

	for _, initEnvFunc := range initEnvFuncs {
		initEnvFunc(env)
	}
	return env
}

var initEnvFuncs []func(inner *Environment)

func InitEnvWith(cb func(*Environment)) {
	initEnvFuncs = append(initEnvFuncs, cb)
}

func IsVirtualTmpDir(ctx context.Context, pa string) bool {
	return strings.HasPrefix(pa, "@tmp/")
}

func GetVirtualTmpDir(ctx context.Context, sessionID interface{}) string {
	if sessionID == nil {
		return "@tmp/"
	}
	return "@tmp/" + as.StringWithDefault(sessionID, "")
}

func GetVirtualDataDir(ctx context.Context, sessionID interface{}) string {
	if sessionID == nil {
		return "@data/"
	}
	return "@data/" + as.StringWithDefault(sessionID, "")
}

func GetVirtualCustomConfDir(ctx context.Context, sessionID interface{}) string {
	if sessionID == nil {
		return "@data/conf/"
	}

	return "@data/conf/" + as.StringWithDefault(sessionID, "")
}

func ToRealDirFunc(env *Environment) func(ctx context.Context, pa string) string {
	return func(ctx context.Context, pa string) string {
		return GetRealDir(ctx, env, pa)
	}
}

func GetRealDir(ctx context.Context, env *Environment, pa string) string {
	if strings.HasPrefix(pa, "@data/conf/") {
		return env.Fs.FromCustomConfig(strings.TrimPrefix(pa, "@data/conf/"))
	}
	if strings.HasPrefix(pa, "@data\\conf\\") {
		return env.Fs.FromCustomConfig(strings.TrimPrefix(pa, "@data\\conf\\"))
	}

	if strings.HasPrefix(pa, "@lib/images/") {
		return env.Fs.FromLib(strings.TrimPrefix(pa, "@lib/images/"))
	}
	if strings.HasPrefix(pa, "@lib\\images\\") {
		return env.Fs.FromLib(strings.TrimPrefix(pa, "@lib\\images\\"))
	}

	if strings.HasPrefix(pa, "@data/") {
		return env.Fs.FromData(strings.TrimPrefix(pa, "@data/"))
	}
	if strings.HasPrefix(pa, "@data\\") {
		return env.Fs.FromData(strings.TrimPrefix(pa, "@data\\"))
	}

	if strings.HasPrefix(pa, "@tmp/") {
		return env.Fs.FromTMP(strings.TrimPrefix(pa, "@tmp/"))
	}
	if strings.HasPrefix(pa, "@tmp\\") {
		return env.Fs.FromTMP(strings.TrimPrefix(pa, "@tmp\\"))
	}
	return pa
}

func NewDownloader(ctx context.Context, env *Environment, rootURL string) (http.Handler, error) {
	if !strings.HasSuffix(rootURL, "/") {
		rootURL = rootURL + "/"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dirname := strings.TrimPrefix(r.URL.Path, rootURL)
		DownloadVFSFile(ctx, env)(ctx, w, r, dirname)
	}), nil
}

func DownloadVFSFile(ctx context.Context, env *Environment) func(ctx context.Context, w http.ResponseWriter, r *http.Request, pa string) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, pa string) {
		pa = path.Clean(strings.TrimPrefix(pa, "/"))
		if !strings.HasPrefix(pa, "@") {
			http.Error(w, "url path is invalid - '"+pa+"'", http.StatusBadRequest)
			return
		}

		filename := GetRealDir(ctx, env, pa)
		if FileExists(filename) {
			http.ServeFile(w, r, filename)
			return
		}
		http.NotFound(w, r)
	}
}

type urlOption int

func (o urlOption) value() urlOption {
	return o
}

const (
	WithHostPort urlOption = iota
	WithoutHostPort
)

type URLOption interface {
	value() urlOption
}

func Urljoin(a, b string) string {
	if strings.HasSuffix(a, "/") {
		if strings.HasPrefix(b, "/") {
			return a + b[1:]
		}
		return a + b
	}
	if strings.HasPrefix(b, "/") {
		return a + b
	}
	return a + "/" + b
}

func NewLogger(namespace string, cfg *Config, fs FileSystem, levelVar *slog.LevelVar) *slog.Logger {
	var out io.Writer
	if filename := cfg.StringWithDefault("log.filename", "boo.log"); filename == "console" {
		out = os.Stderr
	} else if filename == "stdout" {
		out = os.Stdout
	} else if filename == "stderr" {
		out = os.Stderr
	} else {
		out = &lumberjack.Logger{
			Filename:   fs.FromLogDir(filename),
			MaxSize:    cfg.IntWithDefault("log.maxsize", 5),
			MaxAge:     cfg.IntWithDefault("log.maxage", 1),
			MaxBackups: cfg.IntWithDefault("log.max_backups", 5),
			LocalTime:  cfg.BoolWithDefault("log.local_time", true),
			Compress:   cfg.BoolWithDefault("log.compress", true),
		}
	}
	var programLevel slog.Leveler
	if levelVar != nil {
		programLevel = levelVar
	}
	levelStr := cfg.StringWithDefault("log.level", "")
	unkownLevel := false

	switch levelStr {
	case "debug":
		if levelVar == nil {
			programLevel = slog.LevelDebug
		} else {
			levelVar.Set(slog.LevelDebug)
		}
	case "warn":
		if levelVar == nil {
			programLevel = slog.LevelWarn
		} else {
			levelVar.Set(slog.LevelWarn)
		}
	case "error":
		if levelVar == nil {
			programLevel = slog.LevelError
		} else {
			levelVar.Set(slog.LevelError)
		}
	case "info", "":
		if levelVar == nil {
			programLevel = slog.LevelInfo
		} else {
			levelVar.Set(slog.LevelInfo)
		}
	default:
		unkownLevel = true
		if levelVar == nil {
			programLevel = slog.LevelInfo
		} else {
			levelVar.Set(slog.LevelInfo)
		}
	}

	h := slog.NewJSONHandler(out,
		&slog.HandlerOptions{Level: programLevel})
	logger := slog.New(h)
	if unkownLevel {
		logger.Warn("log level is invalid", slog.String("value", levelStr))
	}
	return logger
}
