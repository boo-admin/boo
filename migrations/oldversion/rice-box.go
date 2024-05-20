package oldversion

import (
	"time"

	"github.com/GeertJohan/go.rice/embedded"
)

func init() {

	// define files
	file3 := &embedded.EmbeddedFile{
		Filename:    "oldversion/main.go",
		FileModTime: time.Unix(1716213300, 0),
		Content:     string("package oldversion\n\nimport (\n\t\"io/fs\"\n\n\trice \"github.com/GeertJohan/go.rice\"\n)\n\nfunc GetStaticDir() (fs.FS, error) {\n\tstaticFS, err := rice.FindBox(\"../../migrations\")\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn riceFS{staticFS}, nil\n}\n\ntype riceFS struct {\n\t*rice.Box\n}\n\nfunc (rfs riceFS) Open(name string) (fs.File, error) {\n\treturn rfs.Box.Open(name)\n}\n"),
	}
	file4 := &embedded.EmbeddedFile{
		Filename:    "oldversion/rice-box.go",
		FileModTime: time.Unix(1716213410, 0),
		Content:     string(""),
	}
	file6 := &embedded.EmbeddedFile{
		Filename:    "postgres/20240429111914_add_departments.sql",
		FileModTime: time.Unix(1716179938, 0),
		Content:     string("-- +goose Up\r\n-- +goose StatementBegin\r\nCREATE TABLE IF NOT EXISTS boo_departments (\r\n  id                          bigserial PRIMARY KEY,\r\n  parent_id                   int NULL REFERENCES boo_departments(id) on delete set null,\r\n  uuid                        VARCHAR(50),\r\n  name                        VARCHAR(50),\r\n  order_num                   int,\r\n  created_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),\r\n  updated_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),\r\n  unique(name)\r\n);\r\n-- +goose StatementEnd\r\n\r\n-- +goose Down\r\nDROP TABLE IF EXISTS boo_departments;\r\n"),
	}
	file7 := &embedded.EmbeddedFile{
		Filename:    "postgres/20240429111931_add_users.sql",
		FileModTime: time.Unix(1715416788, 0),
		Content:     string("-- +goose Up\r\n-- +goose StatementBegin\r\nCREATE TABLE IF NOT EXISTS boo_users (\r\n  id                          bigserial PRIMARY KEY,\r\n  department_id               bigint NULL REFERENCES boo_departments(id) on delete set null,\r\n  name                        varchar(100) NOT NULL UNIQUE,\r\n  nickname                    varchar(100) NOT NULL UNIQUE,\r\n  password                    varchar(500) ,\r\n  last_password_modified_at   timestamp WITH TIME ZONE,\r\n  description                 text,\r\n  disabled                    boolean,\r\n  source                      varchar(50),\r\n  fields                      jsonb,\r\n  created_at                  timestamp WITH TIME ZONE,\r\n  updated_at                  timestamp WITH TIME ZONE\r\n);\r\n-- +goose StatementEnd\r\n\r\n-- +goose StatementBegin\r\nCREATE TABLE IF NOT EXISTS boo_user_profiles (\r\n    user_id     bigint REFERENCES boo_users ON DELETE CASCADE,\r\n    name        varchar(100) NOT NULL,\r\n    value       text,\r\n    created_at  TIMESTAMP WITH TIME ZONE,\r\n    updated_at  TIMESTAMP WITH TIME ZONE,\r\n\r\n    UNIQUE(user_id,name)\r\n);\r\n-- +goose StatementEnd\r\n\r\n-- +goose Down\r\nDROP TABLE IF EXISTS boo_user_profiles;\r\nDROP TABLE IF EXISTS boo_users;\r\n"),
	}
	file8 := &embedded.EmbeddedFile{
		Filename:    "postgres/20240429111936_add_oplog.sql",
		FileModTime: time.Unix(1715320751, 0),
		Content:     string("-- +goose Up\r\n-- +goose StatementBegin\r\nCREATE TABLE IF NOT EXISTS boo_operation_logs (\r\n  id           bigserial PRIMARY KEY,\r\n  userid       bigint REFERENCES boo_users(id) ON DELETE SET NULL,\r\n  username     varchar(100),\r\n  type         varchar(100),\r\n  successful   boolean,\r\n  content      text,\r\n  fields       jsonb,\r\n  created_at   timestamp with time zone\r\n);\r\n-- +goose StatementEnd\r\n\r\n-- +goose Down\r\nDROP TABLE IF EXISTS boo_operation_logs;\r\n"),
	}
	file9 := &embedded.EmbeddedFile{
		Filename:    "postgres/20240510093318_add_employees.sql",
		FileModTime: time.Unix(1715416423, 0),
		Content:     string("-- +goose Up\n-- +goose StatementBegin\nCREATE TABLE IF NOT EXISTS boo_employees (\n  id                          bigserial PRIMARY KEY,\n  department_id               bigint NULL REFERENCES boo_departments(id) on delete set null,\n  user_id                     bigint NULL REFERENCES boo_users(id) on delete set null,\n  name                        varchar(100) NOT NULL UNIQUE,\n  nickname                    varchar(100) NOT NULL UNIQUE,\n  description                 text,\n  disabled                    boolean,\n  source                      varchar(50),\n  fields                      jsonb,\n  created_at                  timestamp WITH TIME ZONE,\n  updated_at                  timestamp WITH TIME ZONE\n);\n-- +goose StatementEnd\n\n-- +goose Down\nDROP TABLE IF EXISTS boo_employees;\n"),
	}

	// define dirs
	dir1 := &embedded.EmbeddedDir{
		Filename:   "",
		DirModTime: time.Unix(1716213255, 0),
		ChildFiles: []*embedded.EmbeddedFile{},
	}
	dir2 := &embedded.EmbeddedDir{
		Filename:   "oldversion",
		DirModTime: time.Unix(1716213255, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file3, // "oldversion/main.go"
			file4, // "oldversion/rice-box.go"

		},
	}
	dir5 := &embedded.EmbeddedDir{
		Filename:   "postgres",
		DirModTime: time.Unix(1715334929, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file6, // "postgres/20240429111914_add_departments.sql"
			file7, // "postgres/20240429111931_add_users.sql"
			file8, // "postgres/20240429111936_add_oplog.sql"
			file9, // "postgres/20240510093318_add_employees.sql"

		},
	}

	// link ChildDirs
	dir1.ChildDirs = []*embedded.EmbeddedDir{
		dir2, // "oldversion"
		dir5, // "postgres"

	}
	dir2.ChildDirs = []*embedded.EmbeddedDir{}
	dir5.ChildDirs = []*embedded.EmbeddedDir{}

	// register embeddedBox
	embedded.RegisterEmbeddedBox(`../../migrations`, &embedded.EmbeddedBox{
		Name: `../../migrations`,
		Time: time.Unix(1716213255, 0),
		Dirs: map[string]*embedded.EmbeddedDir{
			"":           dir1,
			"oldversion": dir2,
			"postgres":   dir5,
		},
		Files: map[string]*embedded.EmbeddedFile{
			"oldversion/main.go":                          file3,
			"oldversion/rice-box.go":                      file4,
			"postgres/20240429111914_add_departments.sql": file6,
			"postgres/20240429111931_add_users.sql":       file7,
			"postgres/20240429111936_add_oplog.sql":       file8,
			"postgres/20240510093318_add_employees.sql":   file9,
		},
	})
}
