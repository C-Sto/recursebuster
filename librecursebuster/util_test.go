package librecursebuster

import (
	"fmt"
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
