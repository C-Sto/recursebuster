package testserver

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
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
/appendslash/

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
/c

*/

const bod404 = `404 not found 20/20/19`
const bod404mod = `404 not found 20/20/20`
const bod200 = `200ish response! This should be different enough that it is not detected as being a soft 404, ideally anyway.`

func handler(w http.ResponseWriter, r *http.Request) {

	vMut.Lock()
	//if it's been visited more than once, instant fail
	if visited[r.Method+":"+r.URL.Path] && r.URL.Path != "/" {
		//we can visit the base url more than once
		tes.Fail()
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
	case "/appendslash/":
		fallthrough
	case "/badheader/":
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
		fallthrough
	case "/a/b/c/basicauth":
		respCode = 401
	case "/a/b/c/d":
		respCode = 403
	case "/c":
		fallthrough
	case "/c/":
		fallthrough
	case "/c/badcode":
		respCode = 500
	case "/c/d":
		respCode = 666
	case "/ajaxonly":
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			respCode = 200
		} else {
			respCode = 404
		}
	case "/onlynoajax":
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			respCode = 404
		} else {
			respCode = 200
		}
	case "/ajaxpost":
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" &&
			r.Method == "POST" {
			respCode = 200
		} else {
			respCode = 400
		}
	default:
		respCode = 404
	}
	bod := bod404
	if respCode == 200 {
		bod = bod200
	} else if strings.ToLower(string(r.URL.Path[len(r.URL.Path)-1])) == "x" {
		//404 body
		bod = bod404
	} else if strings.ToLower(string(r.URL.Path[len(r.URL.Path)-1])) == "y" {
		//modified 404
		bod = bod404mod
	}

	if respCode == 401 {
		if u, p, ok := r.BasicAuth(); strings.ToLower(string(r.URL.Path[len(r.URL.Path)-1])) == "basicauth" &&
			ok && u == "test" && p == "test" {
			respCode = 200
		}
	}

	if respCode == 302 || respCode == 301 {
		if strings.ToLower(r.URL.Path) == "/b" {
			w.Header().Set("Location", "/r/")
		} else if strings.ToLower(r.URL.Path) == "/b/c" {
			w.Header().Set("Location", "/r/b")
		} else if strings.ToLower(r.URL.Path) == "/b/c/" {
			w.Header().Set("Location", "/r/b/c")
		} else if strings.ToLower(r.URL.Path) == "/b/x" {
			w.Header().Set("Location", "/r/x")
		} else if strings.ToLower(r.URL.Path) == "/b/y" {
			w.Header().Set("Location", "/r/y")
		}

		bod = ""
	}
	if strings.ToLower(r.URL.Path) == "/badheader/" {
		w.Header().Add("X-Bad-Header", "test123")
	}
	w.WriteHeader(respCode)
	fmt.Fprintln(w, bod)
	fmt.Println(r.Method, r.URL, respCode, bod)
}

var tes *testing.T

//Start starts the test HTTP server
func Start(port string, finishedTest, setup chan struct{}, t *testing.T) {
	s := http.NewServeMux()
	visited = make(map[string]bool)
	vMut = &sync.RWMutex{}
	tes = t
	s.HandleFunc("/", handler)
	go http.ListenAndServe("127.0.0.1:"+port, s)

	for {
		time.Sleep(time.Second * 1)
		resp, err := http.Get("http://localhost:" + port)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}
		// Reached this point: server is up and running!
		break
	}

	close(setup)
	<-finishedTest //this is an ultra gross hack :(
}

var visited map[string]bool
var vMut *sync.RWMutex
