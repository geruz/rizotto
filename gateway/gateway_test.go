package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const indexPage = "<html>app</html>"

// staticGateway serves a small web build from / next to one API route.
func staticGateway(t *testing.T) HTTPGateway {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(indexPage), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o600))

	gt := NewHTTPGateway().Static("/", dir)
	gt.router.HandleFunc("GET /api/v1/me", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("me"))
	})

	return gt
}

func get(t *testing.T, gt HTTPGateway, path string, accept string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)

	if accept != "" {
		request.Header.Set("Accept", accept)
	}

	gt.router.ServeHTTP(recorder, request)

	return recorder
}

func TestStaticServesTheFilesOfTheBuild(t *testing.T) {
	t.Parallel()

	gt := staticGateway(t)

	root := get(t, gt, "/", "text/html")
	assert.Equal(t, http.StatusOK, root.Code)
	assert.Equal(t, indexPage, root.Body.String())

	asset := get(t, gt, "/assets/app.js", "")
	assert.Equal(t, http.StatusOK, asset.Code)
	assert.Equal(t, "console.log(1)", asset.Body.String())
}

func TestStaticAnswersAPageOfTheAppWithIndex(t *testing.T) {
	t.Parallel()

	recorder := get(t, staticGateway(t), "/items/5", "text/html,application/xhtml+xml")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, indexPage, recorder.Body.String())
}

func TestStaticKeepsNotFoundForAssetsAndAPIClients(t *testing.T) {
	t.Parallel()

	gt := staticGateway(t)

	assert.Equal(t, http.StatusNotFound, get(t, gt, "/assets/missing.js", "text/html").Code)
	assert.Equal(t, http.StatusNotFound, get(t, gt, "/api/v1/unknown", "application/json").Code)
	assert.Equal(t, http.StatusNotFound, get(t, gt, "/assets/", "").Code)
	assert.Equal(t, http.StatusNotFound, get(t, gt, "/etc/passwd", "").Code)
}

func TestStaticLeavesTheAPIRoutesAlone(t *testing.T) {
	t.Parallel()

	recorder := get(t, staticGateway(t), "/api/v1/me", "text/html")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "me", recorder.Body.String())
}
