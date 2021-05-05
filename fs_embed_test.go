package render

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed fixtures/*/*.html fixtures/*/*.tmpl fixtures/*/*/*.tmpl fixtures/*/*.amber fixtures/*/*/*.amber
var EmbedFixtures embed.FS

func TestEmbedFileSystemTemplateLookup(t *testing.T) {
	baseDir := "fixtures/template-dir-test"
	fname0Rel := "0"
	fname1Rel := "subdir/1"
	fnameShouldParsedRel := "dedicated.tmpl/notbad"
	dirShouldNotParsedRel := "dedicated"

	r := New(Options{
		Directory:  baseDir,
		Extensions: []string{".tmpl", ".html"},
		FileSystem: &EmbedFileSystem{
			FS: EmbedFixtures,
		},
	})

	expect(t, r.TemplateLookup(fname1Rel) != nil, true)
	expect(t, r.TemplateLookup(fname0Rel) != nil, true)
	expect(t, r.TemplateLookup(fnameShouldParsedRel) != nil, true)
	expect(t, r.TemplateLookup(dirShouldNotParsedRel) == nil, true)
}

func TestEmbedFileSystemHTMLBasic(t *testing.T) {
	render := New(Options{
		Directory: "fixtures/basic",
		FileSystem: &EmbedFileSystem{
			FS: EmbedFixtures,
		},
	})

	var err error
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "hello", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "<h1>Hello gophers</h1>\n")
}
