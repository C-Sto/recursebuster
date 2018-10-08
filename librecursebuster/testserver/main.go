package testserver

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

/*wordlist
a
b
c
d
x
y
*/

/*

200 (OK)
/
/a
/a/b
/a/b/c
/a/
/a/x (200, but same body as /x (404))
/a/y (200, but very similar body to /x (404))

300
/b -> /a/ (302)
/b/c -> /a/b (301)
/b/c/ -> /a/b/c (302)
/b/x (302, but same body as /x (404))
/b/y (301, but very similar body to /x (404))

400
/x (404)
/a/b/c/ (401)
/a/b/c/d (403)

500
/c/d
/c/

*/

const bod404 = `404 not found 20/20/19`
const bod404mod = `404 not found 20/20/20`
const bod200 = `200ish response! This should be different enough that it is not detected as being a soft 404, ideally anyway.`

func handler(w http.ResponseWriter, r *http.Request) {

	vMut.Lock()
	//if it's been visited more than once, instant fail
	if visited[r.Method+":"+r.URL.Path] && r.URL.Path != "/" {
		//we can visit the base url more than once
		panic("Path visited more than once: " + r.Method + ":" + r.URL.Path)
	}
	visited[r.Method+":"+r.URL.Path] = true
	vMut.Unlock()

	respCode := 404

	switch strings.ToLower(r.URL.Path) {
	case "/":
		fallthrough
	case "/a":
		fallthrough
	case "/a/":
		fallthrough
	case "/a/b":
		fallthrough
	case "/a/b/c":
		fallthrough
	case "/a/x":
		fallthrough
	case "/a/y":
		respCode = 200
	case "/b":
		fallthrough
	case "/b/c/":
		fallthrough
	case "/b/x":
		respCode = 302
	case "/b/c":
		fallthrough
	case "/b/y":
		respCode = 301
	case "/x":
		fallthrough
	case "/y":
		respCode = 404
	case "/a/b/c/":
		respCode = 401
	case "/a/b/c/d":
		respCode = 403
	case "/c":
		respCode = 500
	case "/c/d":
		respCode = 666
	default:
		respCode = 404
	}
	w.Header().Set("Location", "cats")

	w.WriteHeader(respCode)
	bod := bod404
	if strings.ToLower(string(r.URL.Path[len(r.URL.Path)-1])) == "x" {
		//404 body
		bod = bod404
	} else if strings.ToLower(r.URL.Path[1:]) == "y" {
		//modified 404
		bod = bod404mod
	} else if respCode == 302 || respCode == 301 {
		bod = ""
	} else if respCode == 200 {
		bod = bod200
	}

	fmt.Fprintln(w, bod)
	/*
		var keys []string
		for k := range r.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fmt.Fprintln(w, "<b>Request Headers:</b></br>", r.URL.Path[1:])
		for _, k := range keys {
			fmt.Fprintln(w, k, ":", r.Header[k], "</br>", r.URL.Path[1:])
		}*/
}

func Start() {
	visited = make(map[string]bool)
	vMut = &sync.RWMutex{}
	http.HandleFunc("/", handler)
	go http.ListenAndServe("127.0.0.1:12345", nil)
}

var visited map[string]bool
var vMut *sync.RWMutex
