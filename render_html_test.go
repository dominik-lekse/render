package render

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTMLBad(t *testing.T) {
	var err error

	render, err := New(Options{
		FileSystem: LocalFS("testdata/basic"),
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "nope", nil)
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNotNil(t, err)
	expect(t, res.Code, 500)
	expect(t, res.Body.String(), "html/template: \"nope\" is undefined\n")
}

func TestHTMLBadDisableHTTPErrorRendering(t *testing.T) {
	var err error

	render, err := New(Options{
		FileSystem:                LocalFS("testdata/basic"),
		DisableHTTPErrorRendering: true,
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "nope", nil)
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNotNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Body.String(), "")
}

func TestHTMLBasic(t *testing.T) {
	var err error

	render, err := New(Options{
		FileSystem: LocalFS("testdata/basic"),
	})
	requireNoError(t, err)

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

func BenchmarkBigHTMLBuffers(b *testing.B) {
	b.ReportAllocs()

	render, err := New(Options{
		FileSystem: LocalFS("testdata/basic"),
	})
	if err != nil {
		b.FailNow()
	}

	var buf = new(bytes.Buffer)
	for i := 0; i < b.N; i++ {
		_ = render.HTML(buf, http.StatusOK, "hello", "gophers")
		buf.Reset()
	}
}

func BenchmarkSmallHTMLBuffers(b *testing.B) {
	b.ReportAllocs()

	render, err := New(Options{
		FileSystem: LocalFS("testdata/basic"),

		// Tiny 8 bytes buffers -> should lead to allocations
		// on every template render
		BufferPool: NewSizedBufferPool(32, 8),
	})
	if err != nil {
		b.FailNow()
	}

	var buf = new(bytes.Buffer)
	for i := 0; i < b.N; i++ {
		_ = render.HTML(buf, http.StatusOK, "hello", "gophers")
		buf.Reset()
	}
}

func TestHTMLXHTML(t *testing.T) {
	var err error

	render, err := New(Options{
		FileSystem:      LocalFS("testdata/basic"),
		HTMLContentType: ContentXHTML,
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "hello", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentXHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "<h1>Hello gophers</h1>\n")
}

func TestHTMLExtensions(t *testing.T) {
	var err error

	render, err := New(Options{
		FileSystem: LocalFS("testdata/basic"),
		Extensions: []string{".tmpl", ".html"},
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "hypertext", nil)
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "Hypertext!\n")
}

func TestHTMLFuncs(t *testing.T) {
	var err error

	render, err := New(Options{
		FileSystem: LocalFS("testdata/custom_funcs"),
		Funcs: []template.FuncMap{{
			"myCustomFunc": func() string {
				return "My custom function"
			},
		}},
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "index", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Body.String(), "My custom function\n")
}

func TestRenderLayout(t *testing.T) {
	var err error

	render, err := New(Options{
		Directory: "testdata/basic",
		Layout:    "layout",
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "content", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Body.String(), "head\n<h1>gophers</h1>\n\nfoot\n")
}

func TestHTMLLayoutCurrent(t *testing.T) {
	var err error

	render, err := New(Options{
		Directory: "testdata/basic",
		Layout:    "current_layout",
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "content", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Body.String(), "content head\n<h1>gophers</h1>\n\ncontent foot\n")
}

func TestHTMLNested(t *testing.T) {
	var err error

	render, err := New(Options{
		Directory: "testdata/basic",
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "admin/index", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "<h1>Admin gophers</h1>\n")
}

func TestHTMLBadPath(t *testing.T) {
	var err error

	render, err := New(Options{
		Directory: "../../../../../../../../../../../../../../../../testdata/basic",
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "hello", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNotNil(t, err)
	expect(t, res.Code, 500)
}

func TestHTMLDelimiters(t *testing.T) {
	var err error

	render, err := New(Options{
		Delims:    Delims{"{[{", "}]}"},
		Directory: "testdata/basic",
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "delims", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "<h1>Hello gophers</h1>")
}

func TestHTMLDefaultCharset(t *testing.T) {
	var err error

	render, err := New(Options{
		Directory: "testdata/basic",
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "hello", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")

	// ContentLength should be deferred to the ResponseWriter and not Render
	expect(t, res.Header().Get(ContentLength), "")
	expect(t, res.Body.String(), "<h1>Hello gophers</h1>\n")
}

func TestHTMLOverrideLayout(t *testing.T) {
	var err error

	render, err := New(Options{
		Directory: "testdata/basic",
		Layout:    "layout",
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "content", "gophers", HTMLOptions{
			Layout: "another_layout",
		})
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "another head\n<h1>gophers</h1>\n\nanother foot\n")
}

func TestHTMLNoRace(t *testing.T) {
	// This test used to fail if run with -race
	render, err := New(Options{
		Directory: "testdata/basic",
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := render.HTML(w, http.StatusOK, "hello", "gophers")
		expectNil(t, err)
	})
	requireNoError(t, err)

	done := make(chan bool)
	doreq := func() {
		res := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)

		h.ServeHTTP(res, req)

		expect(t, res.Code, 200)
		expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")
		// ContentLength should be deferred to the ResponseWriter and not Render
		expect(t, res.Header().Get(ContentLength), "")
		expect(t, res.Body.String(), "<h1>Hello gophers</h1>\n")
		done <- true
	}
	// Run two requests to check there is no race condition
	go doreq()
	go doreq()
	<-done
	<-done
}

func TestHTMLLoadFromAssets(t *testing.T) {
	var err error

	render, err := New(Options{
		FileSystem: &AssetFS{
			Asset: func(file string) ([]byte, error) {
				switch file {
				case "test.tmpl":
					return []byte("<h1>gophers</h1>\n"), nil
				case "layout.tmpl":
					return []byte("head\n{{ yield }}\nfoot\n"), nil
				default:
					return nil, errors.New("file not found: " + file)
				}
			},
			AssetNames: func() []string {
				return []string{"test.tmpl", "layout.tmpl"}
			},
		},
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "test", "gophers", HTMLOptions{
			Layout: "layout",
		})
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 200)
	expect(t, res.Header().Get(ContentType), ContentHTML+"; charset=UTF-8")
	expect(t, res.Body.String(), "head\n<h1>gophers</h1>\n\nfoot\n")
}

func TestCompileTemplatesFromDir(t *testing.T) {
	baseDir := "testdata/template-dir-test"
	fname0Rel := "0"
	fname1Rel := "subdir/1"
	fnameShouldParsedRel := "dedicated.tmpl/notbad"
	dirShouldNotParsedRel := "dedicated"

	r, err := New(Options{
		Directory:  baseDir,
		Extensions: []string{".tmpl", ".html"},
	})
	requireNoError(t, err)

	err = r.CompileTemplates()
	requireNoError(t, err)

	expect(t, r.TemplateLookup(fname1Rel) != nil, true)
	expect(t, r.TemplateLookup(fname0Rel) != nil, true)
	expect(t, r.TemplateLookup(fnameShouldParsedRel) != nil, true)
	expect(t, r.TemplateLookup(dirShouldNotParsedRel) == nil, true)
}

func TestHTMLDisabledCharset(t *testing.T) {
	var err error

	render, err := New(Options{
		Directory:      "testdata/basic",
		DisableCharset: true,
	})
	requireNoError(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.HTML(w, http.StatusOK, "hello", "gophers")
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get(ContentType), ContentHTML)
	expect(t, res.Body.String(), "<h1>Hello gophers</h1>\n")
}
