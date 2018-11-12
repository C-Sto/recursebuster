package librecursebuster

import (
	"fmt"
	"net/url"
	"testing"
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
	urls, e := getUrls([]byte(page))
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
		"/spider",
		"http://localhost.com/spider",
		"../spider",
		"https://localhost:2020/../spider",
		"/spider/../spider",
		"xx://localhost/spider",
		"http://localhost///////////spider",
		"https://localhost:2020/spider",
	}

	for _, x := range cases {
		u, e := url.Parse(x)
		if e != nil {
			t.Error(e)
		}
		fmt.Println(cleanURL(u, "http://localhost"))
	}

	u, e := url.Parse("localhost:2020/spider")
	fmt.Println("Xx", "o"+u.Opaque, "p"+u.Path)

	//u, e = url.Parse("http://localhost:2020/spider")
	//fmt.Println("Xx", "o"+u.Opaque, "p"+u.Path)

	fmt.Println(u, e)

	fmt.Println(cleanURL(u, "http://localhost:2020"))

	t.Error("aaa")
}

/*

func cleanURL(u *url.URL, actualURL string) string {
	fmt.Println(u)
	var didHaveSlash bool
	if len(u.Path) > 0 {
		didHaveSlash = string(u.Path[len(u.Path)-1]) == "/"
		if string(u.Path[0]) != "/" {
			u.Path = "/" + u.Path
		}
	}

	cleaned := path.Clean(u.Path)

	if string(cleaned[0]) != "/" {
		cleaned = "/" + cleaned
	}
	if cleaned != "." {
		actualURL += cleaned
	}

	if didHaveSlash && cleaned != "/" {
		actualURL += "/"
	}
	return actualURL
}
*/
