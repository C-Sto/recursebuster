package librecursebuster

import (
	"bytes"
	"crypto/tls"
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
func ConfigureHTTPClient(sendToBurpOnly bool) *http.Client {

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
		var proxyURL *url.URL
		var err error
		if strings.HasPrefix(gState.Cfg.ProxyAddr, "http") {
			proxyURL, err = url.Parse(gState.Cfg.ProxyAddr)
			if err != nil {
				panic(err)
			}
			httpTransport.Proxy = http.ProxyURL(proxyURL)

		} else {

			dialer, err := proxy.SOCKS5("tcp", gState.Cfg.ProxyAddr, nil, proxy.Direct)
			if err != nil {
				panic(err)
			}
			httpTransport.Dial = dialer.Dial
		}
		//test proxy
		_, err = net.Dial("tcp", proxyURL.Host)
		if err != nil {
			panic(err)
		}
		if !sendToBurpOnly {
			//send the set proxy status (don't need this for burp requests)
			PrintOutput(fmt.Sprintf("Proxy set to: %s", gState.Cfg.ProxyAddr), Info, 0)
		}
	}

	return client
}

//HTTPReq sends the HTTP request based on the given settings, returns the response and the body
//todo: This can probably be optimized to exit once the head has been retreived and discard the body
func HTTPReq(method, path string, client *http.Client) (resp *http.Response, err error) {
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return resp, err
}

func evaluateURL(method string, urlString string, client *http.Client) (headResp *http.Response, content []byte, success bool) {
	success = true
	//wg.Add(1)
	//PrintOutput("EVALUATING:"+method+":"+urlString, Debug, 4, wg, printChan)
	//optimize GET requests by sending a head first (it's cheaper)
	if method == "GET" && !gState.Cfg.NoHead {
		headResp, err := HTTPReq("HEAD", urlString, client) //send a HEAD. Ignore body response
		if err != nil {
			success = false
			<-gState.Chans.workersChan //done with the net thread
			PrintOutput(fmt.Sprintf("%s", err), Error, 0)
			return headResp, content, success
		}

		//Check if we care about it (header only) section
		if gState.BadResponses[headResp.StatusCode] {
			success = false
			<-gState.Chans.workersChan
			return headResp, content, success
		}

		//this is all we have to do if we aren't doing GET's
		if gState.Cfg.NoGet {
			if gState.Cfg.BurpMode { //send successful request again... twice as many requests, but less burp spam
				HTTPReq("HEAD", urlString, gState.BurpClient) //send a HEAD. Ignore body response
			}
			<-gState.Chans.workersChan
			return headResp, content, success
		}
	}

	headResp, err := HTTPReq(method, urlString, client)
	content, _ = ioutil.ReadAll(headResp.Body)
	<-gState.Chans.workersChan //done with the net thread
	if err != nil {
		success = false
		PrintOutput(fmt.Sprintf("%s", err), Error, 0)

		return headResp, content, success
	}

	//Check if we care about it (header only) section
	if gState.BadResponses[headResp.StatusCode] {
		success = false
		return headResp, content, success
	}

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
	//if gState.BadHeaders[headResp.Header.]

	//get content from validated path/file thing
	if gState.Cfg.BurpMode {
		HTTPReq(method, urlString, gState.BurpClient)
	}

	//check we care about it (body only) section
	//double check that it's not 404/error using smart blockchain AI tech
	PrintOutput(
		fmt.Sprintf("Checking body for 404:\nContent: %v,\nSoft404:%v,\nResponse:%v",
			string(content), string(gState.Hosts.Get404Body(headResp.Request.Host)),
			detectSoft404(headResp, gState.Hosts.Get404(headResp.Request.Host), gState.Cfg.Ratio404)),
		Debug, 4)
	if detectSoft404(headResp, gState.Hosts.Get404(headResp.Request.Host), gState.Cfg.Ratio404) {
		//seems to be a soft 404 lol
		return headResp, content, false
	}
	return headResp, content, true
}

//RedirectHandler dictates the way to handle redirects.
func RedirectHandler(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}
