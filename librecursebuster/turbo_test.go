package librecursebuster

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestTurbo(t *testing.T) {
	cl := turboHTTP{}.New()
	r, e := http.NewRequest("GET", "http://www.stealmylogin.com/urlpathhere", nil)
	if e != nil {
		panic(e)
	}
	r2, e := http.NewRequest("GET", "http://www.stealmylogin.com/", nil)
	if e != nil {
		panic(e)
	}
	wg := sync.WaitGroup{}
	wg.Add(5)
	go cl.Do(r, &wg)
	go cl.Do(r2, &wg)
	go cl.Do(r, &wg)
	go cl.Do(r, &wg)
	go cl.Do(r2, &wg)

	time.Sleep(time.Second * 3)
	wg.Wait()
	t.Fail()
}
