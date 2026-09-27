package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

const filesUsage = `Usage:
	rizotto solution add files [flags]

Installs file storage on any S3 compatible object storage: the file service owning
the metadata, the presigner, and the routes handing out upload and download links.

The bytes never travel through the service. A caller asking to upload gets a
presigned URL and sends the file to the storage itself, which is also why no
multipart parsing is needed anywhere.

It is built on the "user" solution: a file belongs to somebody, and every route
checks that the caller owns what they ask for.

Flags:
	-download    how a download is handed over: "redirect" (302 to the storage) or
	             "url" (the link in the json answer) (default "redirect")

The project must have sqlc.yaml (make-project writes it), since the solution owns a table.
`

const (
	downloadRedirect = "redirect"
	downloadURL      = "url"
)

var errDownloadAnswer = errors.New(`-download expects "redirect" or "url"`)

// filesSpec is the data the templates of the files solution are rendered with.
type filesSpec struct {
	Module        string
	RizottoModule string
	// Redirect tells whether a download answers 302 rather than the link itself.
	Redirect bool
	// Methods drive the client interface and the generated bindings.
	Methods []serviceMethod
}

func (f filesSpec) Dir() string {
	return "services/file"
}

func (f filesSpec) ClientImport() string {
	return f.Module + "/" + f.Dir() + "/file-client"
}

func (f filesSpec) APIImport() string {
	return f.Module + "/api"
}

func (f filesSpec) StorageImport() string {
	return f.Module + "/" + f.Dir() + "/storage"
}

func fileMethods() []serviceMethod {
	address := func(action string) string {
		return "http://file-service/file/" + action
	}

	return []serviceMethod{
		{Name: "CreateFileRPC", Request: "CreateFileRequest", Response: "Upload", Address: address("create")},
		{Name: "ConfirmFileRPC", Request: "ConfirmFileRequest", Response: "File", Address: address("confirm")},
		{Name: "GetFileRPC", Request: "GetFileRequest", Response: "File", Address: address("get")},
		{Name: "GetFileLinkRPC", Request: "GetFileLinkRequest", Response: "Link", Address: address("link")},
		{Name: "SelectFilesRPC", Request: "SelectFilesRequest", Response: "FileList", Address: address("select")},
		{Name: "DeleteFileRPC", Request: "DeleteFileRequest", Response: "DeletedFile", Address: address("delete")},
	}
}

type filesSolution struct{}

func (filesSolution) Name() string {
	return filesSolutionName
}

func (filesSolution) Summary() string {
	return "files on S3 compatible storage, handed over as presigned upload and download links"
}

func (filesSolution) Usage() string {
	return filesUsage
}

func (filesSolution) Skill() string {
	return filesSolutionName
}

func (filesSolution) Requires() []string {
	return []string{userSolutionName}
}

func (filesSolution) Marker() string {
	return "services/file/file-service.go"
}

type filesFlags struct {
	download string
}

func (filesSolution) BindFlags(fs *flag.FlagSet) any {
	f := &filesFlags{} //nolint:exhaustruct_v5 // filled by the flag package

	fs.StringVar(&f.download, "download", "",
		`how a download is handed over: "redirect" or "url" (default "redirect")`)

	return f
}

func (filesSolution) Ask(flags any, pr *prompter, prj projectInfo) (solutionPlan, error) {
	f, ok := flags.(*filesFlags)
	if !ok {
		return solutionPlan{}, fmt.Errorf("%w: %T", errWrongFlags, flags)
	}

	if !prj.SQL {
		return solutionPlan{}, errSolutionNeedsSQL
	}

	download, err := answer(pr, f.download,
		"Download as a redirect or as a link in the answer (redirect, url)", downloadRedirect, normalizeDownload)
	if err != nil {
		return solutionPlan{}, err
	}

	spec := filesSpec{
		Module:        prj.Module,
		RizottoModule: rizottoModule,
		Redirect:      download == downloadRedirect,
		Methods:       fileMethods(),
	}

	return solutionPlan{
		Data:         spec,
		Files:        spec.files(),
		Services:     nil,
		Controllers:  nil,
		Repositories: []sqlcEntry{newSqlcEntry(spec.Dir(), "file-queries.sql")},
		Env:          spec.env(),
		Steps:        spec.steps(),
	}, nil
}

func normalizeDownload(raw string) (string, error) {
	answer := strings.ToLower(strings.TrimSpace(raw))
	if answer == downloadRedirect || answer == downloadURL {
		return answer, nil
	}

	return "", fmt.Errorf("%w, got %q", errDownloadAnswer, raw)
}

func (f filesSpec) files() []fileSpec {
	dir := f.Dir()

	return []fileSpec{
		{tmpl: "solutions/files/client.go.tmpl", out: dir + "/file-client/client.go"},
		{tmpl: "solutions/files/bind-gen.go.tmpl", out: dir + "/file-client/bind-gen.go"},
		{tmpl: "solutions/files/file-service.go.tmpl", out: dir + "/file-service.go"},
		{tmpl: "solutions/files/storage.go.tmpl", out: dir + "/storage/s3.go"},
		{tmpl: "solutions/files/storage_test.go.tmpl", out: dir + "/storage/s3_test.go"},
		{tmpl: "solutions/files/repository.go.tmpl", out: dir + "/repository/file-repository.go"},
		{tmpl: "solutions/files/schema.sql.tmpl", out: dir + "/repository/sql/schema.sql"},
		{tmpl: "solutions/files/queries.sql.tmpl", out: dir + "/repository/sql/file-queries.sql"},
		{tmpl: "solutions/files/migration.sql.tmpl", out: dir + "/repository/migrations/000001_create_files.sql"},
		{tmpl: "solutions/files/db/db.go.tmpl", out: dir + "/repository/db/db.go"},
		{tmpl: "solutions/files/db/models.go.tmpl", out: dir + "/repository/db/models.go"},
		{tmpl: "solutions/files/db/queries.sql.go.tmpl", out: dir + "/repository/db/file-queries.sql.go"},
		{tmpl: "solutions/files/file-api.go.tmpl", out: "api/file-api/file.go"},
		{tmpl: "solutions/files/file-api_test.go.tmpl", out: "api/file-api/file_test.go"},
	}
}

func (f filesSpec) env() []envVar {
	return []envVar{
		{
			Key:     "S3_ENDPOINT",
			Value:   "http://localhost:9000",
			Comment: "the storage, without the bucket: https://s3.eu-central-1.amazonaws.com or a MinIO of your own",
		},
		{Key: "S3_REGION", Value: "us-east-1", Comment: ""},
		{Key: "S3_BUCKET", Value: "", Comment: ""},
		{Key: "S3_ACCESS_KEY_ID", Value: "", Comment: ""},
		{Key: "S3_SECRET_ACCESS_KEY", Value: "", Comment: ""},
		{
			Key:     "S3_PATH_STYLE",
			Value:   "true",
			Comment: `"true" puts the bucket in the path, which MinIO needs; AWS wants "false"`,
		},
		{
			Key:     "S3_LINK_TTL_MINUTES",
			Value:   "15",
			Comment: "how long an upload or download link stays valid",
		},
		{
			Key:     "S3_MAX_UPLOAD_MB",
			Value:   "25",
			Comment: "the largest file a caller may announce",
		},
	}
}

func (f filesSpec) steps() []string {
	register := fmt.Sprintf(`register the service in server.go, before the controllers:

         import (
             "%s"
             filesrv "%s/%s"
         )

         file.RegisterServer(filesrv.MustNewFileService(pgPool))`,
		f.ClientImport(), f.Module, f.Dir())

	routes := `join the route table in server.go:

         import "` + f.Module + `/api/file-api"

         gt.Routing(
             gateway.JoinRouteTables(
                 fileapi.NewFileController().RouteTable(),
             ),
         )`

	cors := `allow the browser to PUT straight to the bucket: the upload link points at
     the storage, not at this service, so the bucket needs a CORS rule for
     PUT and GET from the origin of your application.`

	return []string{
		stepEnvExample,
		register,
		routes,
		cors,
		stepGen,
		"task migrate-up # applies the table of the solution",
		"task test       # runs the route tests and writes api/file-api/openapi.yml",
	}
}
