package benchmark

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/c-sto/recursebuster/pkg/net"
)

// https://github.com/OJ/gobuster/blob/master/libgobuster/http_test.go
// similar to this

func httpServerB(b *testing.B, content string) *httptest.Server {
	b.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, content)
	}))
	return ts
}

func BenchmarkGet(b *testing.B) {
	h := httpServerB(b, "test")
	defer h.Close()
	r := net.NewRequester([]byte(""), "test", "", "", "", []string{}, map[string]bool{})
	c := net.ConfigureHTTPClient("", 10, false, false, false, true)
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		_, err := r.HTTPReq("GET", h.URL, c)
		if err != nil {
			b.Fatalf("Got Error: %v", err)
		}
	}
}
