// +build go1.16

package render

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed testdata/*/*.html testdata/*/*.tmpl testdata/*/*/*.tmpl
var EmbedFixtures embed.FS

func TestEmbedFileSystemTemplateLookup(t *testing.T) {
	baseDir := "testdata/template-dir-test"
	fname0Rel := "0"
	fname1Rel := "subdir/1"
	fnameShouldParsedRel := "dedicated.tmpl/notbad"
	dirShouldNotParsedRel := "dedicated"

	fs, err := fs.Sub(EmbedFixtures, baseDir)
	expectNil(t, err)

	r, err := New(Options{
		Extensions: []string{".tmpl", ".html"},
		FileSystem: fs,
	})
	expectNil(t, err)

	expect(t, r.TemplateLookup(fname1Rel) != nil, true)
	expect(t, r.TemplateLookup(fname0Rel) != nil, true)
	expect(t, r.TemplateLookup(fnameShouldParsedRel) != nil, true)
	expect(t, r.TemplateLookup(dirShouldNotParsedRel) == nil, true)
}

func TestEmbedFileSystemHTMLBasic(t *testing.T) {
	var err error

	fs, err := fs.Sub(EmbedFixtures, "fixtures/basic")
	expectNil(t, err)

	render, err := New(Options{
		FileSystem: fs,
	})
	expectNil(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "hello", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "<h1>Hello gophers</h1>\n")
}
