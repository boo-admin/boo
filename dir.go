package boo

// import (
// 	"context"
// 	"errors"
// 	"path/filepath"
// 	"strings"
// )

// func ToRealDirWith(rootDir string) func(ctx context.Context, s string) string {
// 	return ToRealDir(rootDir,
// 		filepath.Join(rootDir, "bin"),
// 		filepath.Join(rootDir, "conf"),
// 		filepath.Join(rootDir, "data/conf"),
// 		filepath.Join(rootDir, "data"),
// 		filepath.Join(rootDir, "lib"),
// 		filepath.Join(rootDir, "data/tmp"))
// }

// func ToRealDir(installDir, binDir, confDir, customconfigDir, dataDir, libDir, tmpDir string) func(ctx context.Context, s string) string {
// 	return func(ctx context.Context, s string) string {
// 		if !strings.HasPrefix(s, "@") {
// 			return s
// 		}
// 		if strings.HasPrefix(s, "@/") {
// 			return filepath.Join(installDir, strings.TrimPrefix(s, "@/"))
// 		}
// 		if strings.HasPrefix(s, "@bin/") {
// 			return filepath.Join(binDir, strings.TrimPrefix(s, "@bin/"))
// 		}
// 		if strings.HasPrefix(s, "@conf/") {
// 			return filepath.Join(confDir, strings.TrimPrefix(s, "@conf/"))
// 		}
// 		if strings.HasPrefix(s, "@data/conf/") {
// 			return filepath.Join(customconfigDir, strings.TrimPrefix(s, "@data/conf/"))
// 		}
// 		if strings.HasPrefix(s, "@data/") {
// 			return filepath.Join(dataDir, strings.TrimPrefix(s, "@data/"))
// 		}
// 		if strings.HasPrefix(s, "@lib/") {
// 			return filepath.Join(libDir, strings.TrimPrefix(s, "@lib/"))
// 		}
// 		if strings.HasPrefix(s, "@tmp/") {
// 			return filepath.Join(tmpDir, strings.TrimPrefix(s, "@tmp/"))
// 		}
// 		panic(errors.New("dir '" + s + "' invalid"))
// 	}
// }
