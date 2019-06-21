package net

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/proxy"
)

//ConfigureHTTPClient configures and returns a HTTP Client (mostly useful to be able to send to burp)
func ConfigureHTTPClient(proxyAddr string, timeout int, burpMode, sendToBurpOnly, followRedirects, ignoreSSl bool) *http.Client {

	httpTransport := &http.Transport{MaxIdleConns: 100}
	client := &http.Client{Transport: httpTransport, Timeout: time.Duration(timeout) * time.Second}

	//skip ssl errors
	httpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: ignoreSSl}

	if !followRedirects {
		client.CheckRedirect = RedirectHandler
	}

	//use a proxy if requested to
	if (proxyAddr != "" && !burpMode) || //proxy is configured, and burpmode is disabled
		(proxyAddr != "" && burpMode && sendToBurpOnly) { // proxy configured, in burpmode, and at the stage where we want to actually send it to burp
		var err error
		if strings.HasPrefix(proxyAddr, "http") {
			var proxyURL *url.URL
			proxyURL, err = url.Parse(proxyAddr)
			if err != nil {
				panic(err)
			}
			httpTransport.Proxy = http.ProxyURL(proxyURL)
			//test proxy
			_, err = net.Dial("tcp", proxyURL.Host)
			if err != nil {
				panic(err)
			}

		} else {

			dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
			if err != nil {
				panic(err)
			}
			httpTransport.Dial = dialer.Dial
			//test proxy
			_, err = net.Dial("tcp", proxyAddr)
			if err != nil {
				panic(err)
			}
		}
		if !sendToBurpOnly {
			//send the set proxy status (don't need this for burp requests)
			//gState.PrintOutput(fmt.Sprintf("Proxy set to: %s", proxyAddr), Info, 0)
		}
	}

	return client
}

type Requester struct {
	body      []byte
	UserAgent string
	Cookies   string
	Auth      string
	Headers   []string
	vHost     string
	blacklist map[string]bool //sorted for faster finding
}

func NewRequester(body []byte, useragent, cookies, auth, vhost string, headers []string, blacklist map[string]bool) *Requester {
	return &Requester{
		body:      body,
		UserAgent: useragent,
		Cookies:   cookies,
		Auth:      auth,
		vHost:     vhost,
		Headers:   headers,
		blacklist: blacklist,
	}

}

//HTTPReq sends the HTTP request based on the given settings, returns the response and the body
//todo: This can probably be optimized to exit once the head has been retreived and discard the body
func (r Requester) HTTPReq(method, path string, client *http.Client) (resp *http.Response, err error) {
	if r.blacklist[path] {
		return nil, errors.New("Blacklisted URL: " + path)
	}

	var req *http.Request
	if len(r.body) > 0 {
		req, err = http.NewRequest(method, path, bytes.NewReader(r.body))
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", r.UserAgent)
	if r.Cookies != "" {
		req.Header.Set("Cookie", r.Cookies)
	}

	if r.Auth != "" {
		req.Header.Set("Authorization", "Basic "+r.Auth)
	}

	if len(r.Headers) > 0 {
		for _, x := range r.Headers {
			spl := strings.Split(x, ":")
			req.Header.Set(spl[0], spl[1])
		}
	}
	if r.vHost != "" {
		req.Host = r.vHost
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//Set body to be readable again
	respBod, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(respBod))
	//when the transfer encoding was chunked - this caused the response length to be -1, breaking the soft 404 stuff. Honestly, the HTTP spec is a massive pain in the ass.
	resp.ContentLength = int64(len(respBod))

	return resp, err
}

//RedirectHandler dictates the way to handle redirects.
func RedirectHandler(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

//DetectSoft404 takes two HTTP responses, and determines how different they are. The response is true if the similarity ratio is above the passed in ratio. The ratio is also returned incase you want to look at it better.
func DetectSoft404(r1 *http.Response, r2 *http.Response, ratio float64) (bool, float64) {
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

//GetURLS returns a slice of strings containing urls
func GetURLs(page []byte) ([]string, error) {

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

//CleanURL will do some lexical cleaning of the URL - removing dots where relevant etc. Takes in the URL object containing the path, and the base to append the path to
func CleanURL(u *url.URL, actualURL string) string {

	if u == nil {
		return actualURL
	}
	//work out if the URL was parsed using an opaque thing (no slash after scheme - probably due to no scheme in link and a port number at the end)
	if u.Opaque != "" {
		//fmt.Println("op", u.String())
		act, err := url.Parse(actualURL)
		if err != nil {
			return actualURL
		}
		u, err = url.Parse(act.Scheme + "://" + u.String())
		if err != nil {
			return actualURL
		}
	}

	var didHaveSlash bool

	//determine if there was a slash at the end before cleaning
	if len(u.Path) > 0 {
		didHaveSlash = string(u.Path[len(u.Path)-1]) == "/"

		//add a slash prefix before the path (to avoid constructions like 'scheme://host:portpath' sneaking in)
		if string(u.Path[0]) != "/" {
			u.Path = "/" + u.Path
		}
	}

	//removes dots and whatnot where it can (so that we don't query for 'host/path/../path/../path/thing' etc)
	cleaned := path.Clean(u.Path)

	//check if the path starts with a / (to avoid that same issue as above with no leading /)
	if string(cleaned[0]) != "/" {
		cleaned = "/" + cleaned
	}
	//check if the cleaned path is just a single dot - if it's not then add the cleaned URL to the url base passed in
	if cleaned != "." && cleaned != "/." {
		actualURL += cleaned
	}

	//check if it had a trailing slash before going in, and if it did, add it back
	if didHaveSlash && !strings.HasSuffix(actualURL, "/") {
		actualURL += "/"
	}
	return actualURL
}
