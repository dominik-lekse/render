package render

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

var ctx = context.Background()

func TestLockConfig(t *testing.T) {
	var err error

	mutex := reflect.TypeOf(&sync.RWMutex{}).Kind()
	empty := reflect.TypeOf(&emptyLock{}).Kind()

	r1, err := New(Options{
		Recompile:    true,
		UseMutexLock: false,
	})
	requireNoError(t, err)
	expect(t, reflect.TypeOf(r1.lock).Kind(), mutex)

	r2, err := New(Options{
		Recompile:    true,
		UseMutexLock: true,
	})
	requireNoError(t, err)
	expect(t, reflect.TypeOf(r2.lock).Kind(), mutex)

	r3, err := New(Options{
		Recompile:    false,
		UseMutexLock: true,
	})
	requireNoError(t, err)
	expect(t, reflect.TypeOf(r3.lock).Kind(), mutex)

	r4, err := New(Options{
		Recompile:    false,
		UseMutexLock: false,
	})
	requireNoError(t, err)
	expect(t, reflect.TypeOf(r4.lock).Kind(), empty)
}

/* Benchmarks */
func BenchmarkNormalJSON(b *testing.B) {
	render, err := New()
	if err != nil {
		b.FailNow()
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = render.JSON(w, 200, Greeting{"hello", "world"})
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(res, req)
	}
}

func BenchmarkStreamingJSON(b *testing.B) {
	render, err := New(Options{
		StreamingJSON: true,
	})
	if err != nil {
		b.FailNow()
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = render.JSON(w, 200, Greeting{"hello", "world"})
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(res, req)
	}
}

func BenchmarkHTML(b *testing.B) {
	render, err := New(Options{
		FileSystem: LocalFS("testdata/basic"),
	})
	if err != nil {
		b.FailNow()
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = render.HTML(w, http.StatusOK, "hello", "gophers")
	})
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
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

func expectNotNil(t *testing.T, a interface{}) {
	if a == nil {
		t.Errorf("Expected ||not nil|| - Got ||nil|| (type %v)", reflect.TypeOf(a))
	}
}

func expectNoError(t *testing.T, err error) bool {
	if err != nil {
		t.Errorf("Expected ||no error|| - Got ||%#v||", err)
		return false
	}

	return true
}

func requireNoError(t *testing.T, err error) {
	if expectNoError(t, err) {
		return
	}
	t.FailNow()
}
