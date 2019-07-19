package test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/c-sto/recursebuster/pkg/net"
)

func TestGetUrls(t *testing.T) {
	urlCount := 6
	page := `	
	<a href=http://test.com/test1 >	
	<a href=../test2.jpg >	
	<a href=test3 >	
	<a href=/cats/cats/cats >	
	<a href='http://test.com/hack/the/planet' >	
	<a href="http://test.com/.git/config" >	
	`
	urls, e := net.GetURLs([]byte(page))
	if e != nil {
		panic(e)
	}
	if len(urls) != urlCount {
		fmt.Println(urls)
		t.Error("Urls not parsed correctly")
	}
}
func TestSoft404Detection(t *testing.T) {
	//todo: write test
}

//bane of my existence
/*
The rawurl may be relative (a path, without a host) or absolute (starting with a scheme). Trying to parse a hostname and path without a scheme is invalid but may not necessarily return an error, due to parsing ambiguities.
*/

func TestCleanURL(t *testing.T) {

	cases := []string{ //all should resolve to a path of 'spider'
		//normal relative/absolute paths
		"/spider",
		"http://localhost.com/spider",
		"../spider",
		"https://localhost:2020/../spider",
		"/spider/../spider",
		"xx://localhost/spider",
		"http://localhost///////////spider",
		"https://localhost:2020/spider",
		"://localhost/spider",
		"//localhost/spider",
		"http://localhost:/spider",
		"://localhost/spider",
		//janky paths that may include a hostname
		//"localhost/spider", //This won't/shouldn't happen / It's kind of hard to think about what the fix for this would be without breaking stuff horribly
	}

	for _, x := range cases {
		//use same method to get around the scheme bug
		if strings.HasPrefix(x, "://") {
			//add a garbage scheme to get past the url parse stuff (the scheme will be added from the reference anyway)
			x = "xxx" + x
		}
		u, e := url.Parse(x)
		if e != nil {
			t.Fatal(e)
		}
		//fmt.Println(x, "H->"+u.Host, "P->"+u.Path)
		result, err := url.Parse((net.CleanURL(u, "http://localhost")))
		if err != nil {
			t.Error(e)
		}
		if result.Path != "/spider" {
			t.Error("Parsed incorrectly: " + x + " -> " + result.Path)
		}
	}
}
