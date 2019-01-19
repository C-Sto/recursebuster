package librecursebuster

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

//ConfigureHTTPClient configures and returns a HTTP Client (mostly useful to be able to send to burp)
func (gState *State) ConfigureHTTPClient(sendToBurpOnly bool) *http.Client {

	httpTransport := &http.Transport{MaxIdleConns: 100}
	client := &http.Client{Transport: httpTransport, Timeout: time.Duration(gState.Cfg.Timeout) * time.Second}

	//skip ssl errors
	httpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: gState.Cfg.SSLIgnore}

	if !gState.Cfg.FollowRedirects {
		client.CheckRedirect = RedirectHandler
	}

	//use a proxy if requested to
	if (gState.Cfg.ProxyAddr != "" && !gState.Cfg.BurpMode) || //proxy is configured, and burpmode is disabled
		(gState.Cfg.ProxyAddr != "" && gState.Cfg.BurpMode && sendToBurpOnly) { // proxy configured, in burpmode, and at the stage where we want to actually send it to burp
		var err error
		if strings.HasPrefix(gState.Cfg.ProxyAddr, "http") {
			var proxyURL *url.URL
			proxyURL, err = url.Parse(gState.Cfg.ProxyAddr)
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

			dialer, err := proxy.SOCKS5("tcp", gState.Cfg.ProxyAddr, nil, proxy.Direct)
			if err != nil {
				panic(err)
			}
			httpTransport.Dial = dialer.Dial
			//test proxy
			_, err = net.Dial("tcp", gState.Cfg.ProxyAddr)
			if err != nil {
				panic(err)
			}
		}
		if !sendToBurpOnly {
			//send the set proxy status (don't need this for burp requests)
			gState.PrintOutput(fmt.Sprintf("Proxy set to: %s", gState.Cfg.ProxyAddr), Info, 0)
		}
	}

	return client
}

//HTTPReq sends the HTTP request based on the given settings, returns the response and the body
//todo: This can probably be optimized to exit once the head has been retreived and discard the body
func (gState *State) HTTPReq(method, path string, client *http.Client) (resp *http.Response, err error) {

	if gState.Blacklist[path] {
		return nil, errors.New("Blacklisted URL: " + path)
	}

	var req *http.Request
	if gState.Cfg.BodyContent != "" {
		req, err = http.NewRequest(method, path, strings.NewReader(gState.bodyContent))
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", gState.Cfg.Agent)
	if gState.Cfg.Cookies != "" {
		req.Header.Set("Cookie", gState.Cfg.Cookies)
	}

	if gState.Cfg.Auth != "" {
		req.Header.Set("Authorization", "Basic "+gState.Cfg.Auth)
	}

	if len(gState.Cfg.Headers) > 0 {
		for _, x := range gState.Cfg.Headers {
			spl := strings.Split(x, ":")
			req.Header.Set(spl[0], spl[1])
		}
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//Set body to be readable again
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return resp, err
}

func (gState *State) evaluateURL(method string, urlString string, client *http.Client) (headResp *http.Response, content []byte, success bool) {

	//optimize GET requests by sending a head first (it's cheaper)
	if method == "GET" && !gState.Cfg.NoHead {
		headResp, err := gState.HTTPReq("HEAD", urlString, client) //send a HEAD. Ignore body response
		if err != nil {
			success = false
			//<-gState.Chans.workersChan //done with the net thread
			gState.PrintOutput(fmt.Sprintf("%s", err), Error, 0)
			return headResp, content, success
		}

		//Check if we care about it (header only) section
		if len(gState.GoodResponses) > 0 {
			if v, ok := gState.GoodResponses[headResp.StatusCode]; !ok || !v {
				success = false
				//<-gState.Chans.workersChan
				return headResp, content, success
			}
		} else {
			if gState.BadResponses[headResp.StatusCode] {
				success = false
				//<-gState.Chans.workersChan
				return headResp, content, success
			}
		}

		//this is all we have to do if we aren't doing GET's
		if gState.Cfg.NoGet {
			success = true
			if gState.Cfg.BurpMode { //send successful request again... twice as many requests, but less burp spam
				gState.HTTPReq("HEAD", urlString, gState.BurpClient) //send a HEAD. Ignore body response
			}
			//<-gState.Chans.workersChan
			return headResp, content, success
		}
	}

	headResp, err := gState.HTTPReq(method, urlString, client)
	if err != nil {
		gState.PrintOutput(fmt.Sprintf("%s", err), Error, 0)
		return headResp, content, false
	}
	content, err = ioutil.ReadAll(headResp.Body)
	if err != nil {
		gState.PrintOutput(fmt.Sprintf("%s", err), Error, 0)
		return headResp, content, false
	}
	headResp.Body = ioutil.NopCloser(bytes.NewBuffer(content))
	//<-gState.Chans.workersChan //done with the net thread
	if err != nil {
		success = false
		gState.PrintOutput(fmt.Sprintf("%s", err), Error, 0)

		return headResp, content, success
	}

	//Check if we care about it (response code only) section
	if len(gState.GoodResponses) > 0 {
		if v, ok := gState.GoodResponses[headResp.StatusCode]; !ok || !v {
			success = false
			//<-gState.Chans.workersChan
			return headResp, content, success
		}
	} else {
		if gState.BadResponses[headResp.StatusCode] {
			success = false
			return headResp, content, success
		}
	}

	//check for bad headers in the response
	if len(gState.Cfg.BadHeader) > 0 {
		for _, x := range gState.Cfg.BadHeader {
			spl := strings.Split(x, ":")
			//check headers for key
			if headResp.Header.Get(spl[0]) != "" {
				//key exists. Check value matches prefix provided in the input
				a := strings.TrimSpace(headResp.Header.Get(spl[0]))
				b := strings.TrimSpace(spl[1])
				if strings.HasPrefix(a, b) || strings.Compare(a, b) == 0 {
					return headResp, content, false
				}
			}
		}
	}

	//check we care about it (body only) section
	//double check that it's not 404/error using smart blockchain AI tech
	is404, _ := detectSoft404(headResp, gState.Hosts.Get404(headResp.Request.Host), gState.Cfg.Ratio404)

	if is404 {
		//seems to be a soft 404 lol
		return headResp, content, false
	}

	if gState.Cfg.BurpMode {
		gState.HTTPReq(method, urlString, gState.BurpClient)
	}
	return headResp, content, true
}

//RedirectHandler dictates the way to handle redirects.
func RedirectHandler(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}
