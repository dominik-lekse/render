package render

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDataBinaryBasic(t *testing.T) {
	var err error

	render, err := New(Options{
		// nothing here to configure
	})
	expectNil(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.Data(w, 299, []byte("hello there"))
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, 299)
	expect(t, res.Header().Get(ContentType), ContentBinary)
	expect(t, res.Body.String(), "hello there")
}

func TestDataCustomMimeType(t *testing.T) {
	var err error

	render, err := New(Options{
		// nothing here to configure
	})
	expectNil(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(ContentType, "image/jpeg")
		err = render.Data(w, http.StatusOK, []byte("..jpeg data.."))
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get(ContentType), "image/jpeg")
	expect(t, res.Body.String(), "..jpeg data..")
}

func TestDataCustomContentType(t *testing.T) {
	var err error

	render, err := New(Options{
		BinaryContentType: "image/png",
	})
	expectNil(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = render.Data(w, http.StatusOK, []byte("..png data.."))
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	h.ServeHTTP(res, req)

	expectNil(t, err)
	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get(ContentType), "image/png")
	expect(t, res.Body.String(), "..png data..")
}
