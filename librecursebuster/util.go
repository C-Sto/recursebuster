package librecursebuster

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"path"

	"github.com/PuerkitoBio/goquery"
)

//RandString will return a UUID
func RandString() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

}

//returns a slice of strings containing urls
func getUrls(page []byte) ([]string, error) {

	ret := []string{}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		return nil, err
	}

	doc.Find("*").Each(func(index int, item *goquery.Selection) {
		linkTag := item
		link, _ := linkTag.Attr("href")
		if len(link) > 0 {
			ret = append(ret, link)
		}
	})

	return ret, nil
}

func detectSoft404(r1 *http.Response, r2 *http.Response, ratio float64) (bool, float64) {
	//a, b := []byte{}
	if r1 != nil && r2 != nil {
		if r1.ContentLength > 0 && r2.ContentLength > 0 { //&&
			//r1.StatusCode == r2.StatusCode {
			a, e := ioutil.ReadAll(r1.Body)
			r1.Body = ioutil.NopCloser(bytes.NewBuffer(a))
			if e != nil {
				panic(e)
			}
			b, e := ioutil.ReadAll(r2.Body)
			r2.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			if e != nil {
				panic(e)
			}
			dist := float64(levenshteinDistance(a, b))
			longer := math.Max(float64(len(a)), float64(len(b)))
			perc := (longer - dist) / longer
			if perc > ratio {
				//if diff.QuickRatio() > ratio {
				return true, perc
			}
		}
	}
	return false, 0
}

//todo: test this is correct
func levenshteinDistance(s []byte, t []byte) int {
	//A+ props to codingo for constantly saying this would be a good idea, eventually I listened I guess?
	//https://gist.github.com/laurent22/8025413 with edits for byte comparisons
	// degenerate cases
	//	s = strings.ToLower(s)
	//	t = strings.ToLower(t)
	if bytes.Compare(s, t) == 0 {
		return 0
	}
	if len(s) == 0 {
		return len(t)
	}
	if len(t) == 0 {
		return len(s)
	}
	if len(s) < len(t) {
		temp := s
		s = t
		t = temp
	}
	// create two work vectors of integer distances
	v0 := make([]int, len(t)+1)
	v1 := make([]int, len(t)+1)

	// initialize v0 (the previous row of distances)
	// this row is A[0][i]: edit distance for an empty s
	// the distance is just the number of characters to delete from t
	for i := 0; i < len(v0); i++ {
		v0[i] = i
	}

	for i := 0; i < len(s); i++ {
		// calculate v1 (current row distances) from the previous row v0

		// first element of v1 is A[i+1][0]
		//   edit distance is delete (i+1) chars from s to match empty t
		v1[0] = i + 1

		// use formula to fill in the rest of the row
		for j := 0; j < len(t); j++ {
			var cost int
			if bytes.EqualFold([]byte{s[i]}, []byte{s[j]}) { // case insensitive hack
				cost = 0
			} else {
				cost = 1
			}
			v1[j+1] = int(math.Min(float64(v1[j]+1), math.Min(float64(v0[j+1]+1), float64(v0[j]+cost))))
		}

		// copy v1 (current row) to v0 (previous row) for next iteration
		for j := 0; j < len(v0); j++ {
			v0[j] = v1[j]
		}
	}

	return v1[len(t)]
}

func cleanURL(u *url.URL, actualURL string) string {
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
