// +build go1.16

package embed

import (
	"github.com/unrolled/render"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestEmbedFileSystemTemplateLookup(t *testing.T) {
	baseDir := "fixtures/template-dir-test"
	fname0Rel := "0"
	fname1Rel := "subdir/1"
	fnameShouldParsedRel := "dedicated.tmpl/notbad"
	dirShouldNotParsedRel := "dedicated"

	r := render.New(render.Options{
		Directory:  baseDir,
		Extensions: []string{".tmpl", ".html"},
		FileSystem: &render.EmbedFileSystem{
			FS: Fixtures,
		},
	})

	expect(t, r.TemplateLookup(fname1Rel) != nil, true)
	expect(t, r.TemplateLookup(fname0Rel) != nil, true)
	expect(t, r.TemplateLookup(fnameShouldParsedRel) != nil, true)
	expect(t, r.TemplateLookup(dirShouldNotParsedRel) == nil, true)
}

func TestEmbedFileSystemHTMLBasic(t *testing.T) {
	r := render.New(render.Options{
		Directory: "fixtures/basic",
		FileSystem: &render.EmbedFileSystem{
			FS: Fixtures,
		},
	})

	var err error
	h := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		err = r.HTML(resp, http.StatusOK, "hello", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(render.ContentType), render.ContentHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "<h1>Hello gophers</h1>\n")
}

/* Test Helper */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected ||%#v|| (type %v) - Got ||%#v|| (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func expectNil(t *testing.T, a interface{}) {
	if a != nil {
		t.Errorf("Expected ||nil|| - Got ||%#v|| (type %v)", a, reflect.TypeOf(a))
	}
}
