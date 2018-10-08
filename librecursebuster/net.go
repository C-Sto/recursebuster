package librecursebuster

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

//ConfigureHTTPClient configures and returns a HTTP Client (mostly useful to be able to send to burp)
func ConfigureHTTPClient(cfg *Config, wg *sync.WaitGroup, printChan chan OutLine, sendToBurpOnly bool) *http.Client {

	httpTransport := &http.Transport{MaxIdleConns: 100}
	client := &http.Client{Transport: httpTransport, Timeout: time.Duration(cfg.Timeout) * time.Second}

	//skip ssl errors
	httpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.SSLIgnore}

	if !cfg.FollowRedirects {
		client.CheckRedirect = RedirectHandler
	}

	//use a proxy if requested to
	if (cfg.ProxyAddr != "" && !cfg.BurpMode) || //proxy is configured, and burpmode is disabled
		(cfg.ProxyAddr != "" && cfg.BurpMode && sendToBurpOnly) { // proxy configured, in burpmode, and at the stage where we want to actually send it to burp
		if strings.HasPrefix(cfg.ProxyAddr, "http") {
			proxyURL, err := url.Parse(cfg.ProxyAddr)
			if err != nil {
				fmt.Println(err)
			}
			httpTransport.Proxy = http.ProxyURL(proxyURL)

		} else {

			dialer, err := proxy.SOCKS5("tcp", cfg.ProxyAddr, nil, proxy.Direct)
			if err != nil {
				os.Exit(1)
			}
			httpTransport.Dial = dialer.Dial
		}
		if !sendToBurpOnly {
			//send the set proxy status (don't need this for burp requests)
			PrintOutput(fmt.Sprintf("Proxy set to: %s", cfg.ProxyAddr), Info, 0, wg, printChan)
		}
	}

	return client
}

//HTTPReq sends the HTTP request based on the given settings, returns the response and the body
//todo: This can probably be optimized to exit once the head has been retreived and discard the body
func HTTPReq(method, path string, client *http.Client, cfg *Config) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, path, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", cfg.Agent)
	if cfg.Cookies != "" {
		req.Header.Set("Cookie", cfg.Cookies)
	}

	if cfg.Auth != "" {
		req.Header.Set("Authorization", "Basic "+cfg.Auth)
	}

	if len(cfg.Headers) > 0 {
		for _, x := range cfg.Headers {
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

func evaluateURL(wg *sync.WaitGroup, cfg *Config, method string, urlString string, client *http.Client, workers chan struct{}, printChan chan OutLine) (headResp *http.Response, content []byte, success bool) {
	success = true
	//wg.Add(1)
	//PrintOutput("EVALUATING:"+method+":"+urlString, Debug, 4, wg, printChan)
	//optimize GET requests by sending a head first (it's cheaper)
	if method == "GET" && !cfg.NoHead {
		headResp, err := HTTPReq("HEAD", urlString, client, cfg) //send a HEAD. Ignore body response
		if err != nil {
			success = false
			<-workers //done with the net thread
			PrintOutput(fmt.Sprintf("%s", err), Error, 0, wg, printChan)
			return headResp, content, success
		}

		//Check if we care about it (header only) section
		if gState.BadResponses[headResp.StatusCode] {
			success = false
			<-workers
			return headResp, content, success
		}

		//this is all we have to do if we aren't doing GET's
		if cfg.NoGet {
			if cfg.BurpMode { //send successful request again... twice as many requests, but less burp spam
				HTTPReq("HEAD", urlString, gState.BurpClient, cfg) //send a HEAD. Ignore body response
			}
			<-workers
			return headResp, content, success
		}
	}

	headResp, err := HTTPReq(method, urlString, client, cfg)
	content, _ = ioutil.ReadAll(headResp.Body)
	<-workers //done with the net thread
	if err != nil {
		success = false
		PrintOutput(fmt.Sprintf("%s", err), Error, 0, wg, printChan)

		return headResp, content, success
	}

	//Check if we care about it (header only) section
	if gState.BadResponses[headResp.StatusCode] {
		success = false
		return headResp, content, success
	}

	if len(cfg.BadHeader) > 0 {
		for _, x := range cfg.BadHeader {
			spl := strings.Split(x, ":")
			if strings.HasPrefix(headResp.Header.Get(spl[0]), strings.Join(spl[1:], ":")) {
				return headResp, content, false
			}
		}
	}
	//if gState.BadHeaders[headResp.Header.]

	//get content from validated path/file thing
	if cfg.BurpMode {
		HTTPReq(method, urlString, gState.BurpClient, cfg)
	}

	//check we care about it (body only) section
	//double check that it's not 404/error using smart blockchain AI tech
	PrintOutput(
		fmt.Sprintf("Checking body for 404:\nContent: %v,\nSoft404:%v,\nResponse:%v",
			string(content), string(gState.Hosts.Get404Body(headResp.Request.Host)),
			detectSoft404(headResp, gState.Hosts.Get404(headResp.Request.Host), cfg.Ratio404)),
		Debug, 4, wg, printChan)
	if detectSoft404(headResp, gState.Hosts.Get404(headResp.Request.Host), cfg.Ratio404) {
		//seems to be a soft 404 lol
		return headResp, content, false
	}
	return headResp, content, true
}

//RedirectHandler dictates the way to handle redirects.
func RedirectHandler(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}
