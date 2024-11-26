package booclient

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Returns same path as Executable, returns just the folder
// path. Excludes the executable name and any trailing slash.
func ExecutableFolder() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Dir(p), nil
}

// FileSystem 运行环境中文件系统的抽象
type FileSystem interface {
	FromRun(s ...string) string
	FromInstallRoot(s ...string) string
	// FromWebConfig(s ...string) string
	// FromBin(s ...string) string
	FromLib(s ...string) string
	FromRuntimeEnv(s ...string) string
	FromData(s ...string) string
	FromTMP(s ...string) string
	FromConfig(s ...string) string
	FromLogDir(s ...string) string
	FromCustomConfig(s ...string) string
	FromSession(s ...string) string
	SearchConfig(s ...string) []string
}

type linuxFs struct {
	installDir  string
	binDir      string
	logDir      string
	dataDir     string
	dataConfDir string
	tmpDir      string
	runDir      string
}

func (fs *linuxFs) FromInstallRoot(s ...string) string {
	return filepath.Join(fs.installDir, filepath.Join(s...))
}

func (fs *linuxFs) FromRun(s ...string) string {
	return filepath.Join(fs.runDir, filepath.Join(s...))
}

// func (fs *linuxFs) FromWebConfig(s ...string) string {
// 	return filepath.Join(fs.dataConfDir, "web", filepath.Join(s...))
// }

func (fs *linuxFs) Set(key, s string) {
	switch key {
	case "install":
		fs.installDir = s
	case "bin":
		fs.binDir = s
	case "log":
		fs.logDir = s
	case "data":
		fs.dataDir = s
	// case "conf":
	// 	fs.confDir = s
	case "custom_conf":
		fs.dataConfDir = s
	case "tmp":
		fs.tmpDir = s
	case "run":
		fs.runDir = s
	}
}

func (fs *linuxFs) FromBin(s ...string) string {
	return filepath.Join(fs.binDir, filepath.Join(s...))
}

func (fs *linuxFs) FromLib(s ...string) string {
	return filepath.Join(fs.installDir, "lib", filepath.Join(s...))
}

func (fs *linuxFs) FromRuntimeEnv(s ...string) string {
	return filepath.Join(fs.installDir, "runtime_env", filepath.Join(s...))
}

func (fs *linuxFs) FromData(s ...string) string {
	return filepath.Join(fs.dataDir, filepath.Join(s...))
}

func (fs *linuxFs) FromTMP(s ...string) string {
	return filepath.Join(fs.tmpDir, filepath.Join(s...))
}

func (fs *linuxFs) FromConfig(s ...string) string {
	return filepath.Join(fs.installDir, "conf", filepath.Join(s...))
}

func (fs *linuxFs) FromCustomConfig(s ...string) string {
	return filepath.Join(fs.dataConfDir, filepath.Join(s...))
}

func (fs *linuxFs) FromLogDir(s ...string) string {
	return filepath.Join(fs.logDir, filepath.Join(s...))
}

func (fs *linuxFs) FromSession(s ...string) string {
	return filepath.Join(fs.tmpDir, "sessions", filepath.Join(s...))
}

func (fs *linuxFs) SearchConfig(s ...string) []string {
	var files []string
	for _, nm := range []string{
		fs.FromConfig(filepath.Join(s...)),
		fs.FromCustomConfig(filepath.Join(s...)),
	} {
		if st, err := os.Stat(nm); err == nil && !st.IsDir() {
			files = append(files, nm)
		} else if err != nil && os.IsPermission(err) {
			panic(err)
		}
	}
	return files
}

func NewFileSystem(namespace string, params map[string]string) (*linuxFs, error) {
	var fs *linuxFs

	var rootDir = os.Getenv(namespace + "_root_dir")
	if params != nil {
		if s := params[namespace+"_root_dir"]; s != "" {
			rootDir = s
		}
	}

	isWindows := runtime.GOOS == "windows"
	if isWindows {
		if rootDir == "<default>" || rootDir == "." {
			// "<default>" 作为一个特殊的字符，自动使用当前目录
			if cwd, e := os.Getwd(); nil == e {
				rootDir = cwd
			} else {
				rootDir = "."
			}
		}

		if rootDir == "" {
			exeDir, _ := ExecutableFolder()

			for _, filename := range []string{
				filepath.Join("conf", "app.properties"),
				filepath.Join("..", "conf", "app.properties"),

				filepath.Join(exeDir, "conf", "app.properties"),
				filepath.Join(exeDir, "..", "conf", "app.properties"),
			} {
				if abs, err := filepath.Abs(filename); err == nil {
					filename = abs
				}

				if st, err := os.Stat(filename); err == nil && !st.IsDir() {
					rootDir = filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
					break
				} else if os.IsPermission(err) {
					return nil, err
				}
			}
		}
		if rootDir == "" {
			for _, s := range filepath.SplitList(os.Getenv("GOPATH")) {
				abs, _ := filepath.Abs(filepath.Join(s, "src/cn/com/hengwei"))
				abs = filepath.Clean(abs)
				if st, err := os.Stat(abs); err == nil && st.IsDir() {
					rootDir = abs
					break
				} else if err != nil && os.IsPermission(err) {
					panic(err)
				}
			}
		}
		if rootDir == "" {
			if cwd, e := os.Getwd(); nil == e {
				rootDir = cwd
			} else {
				rootDir = "."
			}
		}

		fs = &linuxFs{
			installDir:  rootDir,
			binDir:      filepath.Join(rootDir, "bin"),
			logDir:      filepath.Join(rootDir, "logs"),
			dataDir:     filepath.Join(rootDir, "data"),
			dataConfDir: filepath.Join(rootDir, "data", "conf"),
			tmpDir:      filepath.Join(rootDir, "data", "tmp"),
			runDir:      rootDir,
		}
	} else {
		if strings.HasPrefix(rootDir, "/opt/") {
			fs = &linuxFs{
				installDir:  filepath.Join(rootDir, "/install"),
				binDir:      filepath.Join(rootDir, "/install/bin"),
				logDir:      filepath.Join(rootDir, "/logs"),
				dataDir:     filepath.Join(rootDir, "/data"),
				dataConfDir: filepath.Join(rootDir, "/conf"),
				tmpDir:      filepath.Join(rootDir, "/tmp"),
				runDir:      filepath.Join(rootDir, "/run"),
			}
		} else {
			fs = &linuxFs{
				installDir:  "/usr/local/" + namespace,
				binDir:      "/usr/local/" + namespace + "/bin",
				logDir:      "/var/log/" + namespace,
				dataDir:     "/var/lib/" + namespace,
				dataConfDir: "/etc/" + namespace,
				tmpDir:      filepath.Join(os.TempDir(), namespace),
				runDir:      "/var/run/" + namespace,
			}
		}
	}

	if installDir := os.Getenv(namespace + "_install_dir"); installDir != "" {
		fs.installDir = installDir
		fs.binDir = filepath.Join(installDir, "bin")
	}
	if dataDir := os.Getenv(namespace + "_data_dir"); dataDir != "" {
		fs.dataDir = dataDir
		if isWindows {
			fs.dataConfDir = filepath.Join(fs.dataDir, "conf")
			fs.tmpDir = filepath.Join(fs.dataDir, "tmp")
		}
	}
	if confDir := os.Getenv(namespace + "_custom_conf_dir"); confDir != "" {
		fs.dataConfDir = confDir
	}
	if logDir := os.Getenv(namespace + "_log_dir"); logDir != "" {
		fs.logDir = logDir
	}
	if tmpDir := os.Getenv(namespace + "_tmp_dir"); tmpDir != "" {
		fs.tmpDir = tmpDir
	}
	if runDir := os.Getenv(namespace + "_run_dir"); runDir != "" {
		fs.runDir = runDir
	}

	if params != nil {
		initFsWithConfig(fs, namespace, params)
	}

	return fs, nil
}

func initFsWithConfig(filesystem FileSystem, namespace string, params map[string]string) {
	fs := filesystem.(*linuxFs)
	if s := params[namespace+"_install_dir"]; s != "" {
		fs.Set("install", s)
		fs.Set("bin", filepath.Join(s, "bin"))
	}
	if s := params[namespace+"_data_dir"]; s != "" {
		fs.Set("data", s)
		if isWindows {
			fs.Set("custom_conf", filepath.Join(fs.dataDir, "conf"))
			fs.Set("tmp", filepath.Join(fs.dataDir, "tmp"))
		}
	}
	if s := params[namespace+"_custom_conf_dir"]; s != "" {
		fs.Set("custom_conf", s)
	}
	if logDir := params[namespace+"_log_dir"]; logDir != "" {
		fs.Set("log", logDir)
	}
	if tmpDir := params[namespace+"_tmp_dir"]; tmpDir != "" {
		fs.Set("tmp", tmpDir)
	}
	if runDir := params[namespace+"_run_dir"]; runDir != "" {
		fs.Set("run", runDir)
	}
}
