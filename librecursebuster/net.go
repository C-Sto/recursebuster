package librecursebuster

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

func ConfigureHTTPClient(cfg Config, wg *sync.WaitGroup, printChan chan OutLine, sendToBurpOnly bool) *http.Client {

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
			proxyUrl, err := url.Parse(cfg.ProxyAddr)
			if err != nil {
				fmt.Println(err)
			}
			httpTransport.Proxy = http.ProxyURL(proxyUrl)

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

func HttpReq(method, path string, client *http.Client, cfg Config) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, path, nil)

	if err != nil {
		return nil, nil, err
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	buf := &bytes.Buffer{}
	buf.ReadFrom(resp.Body)
	body := buf.Bytes()

	return resp, body, err
}

func evaluateURL(wg *sync.WaitGroup, cfg Config, state State, method string, urlString string, client *http.Client, workers chan struct{}, printChan chan OutLine) (headResp *http.Response, content []byte, success bool) {
	success = true

	//optimize GET requests by sending a head first (it's cheaper)
	if method == "GET" && !cfg.NoHead {
		headResp, _, err := HttpReq("HEAD", urlString, client, cfg) //send a HEAD. Ignore body response
		if err != nil {
			success = false
			<-workers //done with the net thread
			PrintOutput(fmt.Sprintf("%s", err), Error, 0, wg, printChan)
			return headResp, content, success
		}

		//Check if we care about it (header only) section
		if state.BadResponses[headResp.StatusCode] {
			success = false
			<-workers
			return headResp, content, success
		}

		//this is all we have to do if we aren't doing GET's
		if cfg.NoGet {
			if cfg.BurpMode { //send successful request again... twice as many requests, but less burp spam

				client = ConfigureHTTPClient(cfg, wg, printChan, true)
				HttpReq("HEAD", urlString, client, cfg) //send a HEAD. Ignore body response
			}
			<-workers
			return headResp, content, success
		}
	}

	headResp, content, err := HttpReq(method, urlString, client, cfg)
	<-workers //done with the net thread
	if err != nil {
		success = false
		PrintOutput(fmt.Sprintf("%s", err), Error, 0, wg, printChan)

		return headResp, content, success
	}

	//Check if we care about it (header only) section
	if state.BadResponses[headResp.StatusCode] {
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
	//if state.BadHeaders[headResp.Header.]

	//get content from validated path/file thing
	if cfg.BurpMode {
		client = ConfigureHTTPClient(cfg, wg, printChan, true)
		HttpReq(method, urlString, client, cfg)
	}

	//check we care about it (body only) section
	//double check that it's not 404/error using smart blockchain AI tech
	PrintOutput(
		fmt.Sprintf("Checking body for 404:\nContent: %v,\nSoft404:%v,\nResponse:%v",
			string(content), string(state.Hosts.Get404Body(headResp.Request.Host)),
			detectSoft404(content, state.Hosts.Get404Body(headResp.Request.Host), cfg.Ratio404)),
		Debug, 4, wg, printChan)
	if detectSoft404(content, state.Hosts.Get404Body(headResp.Request.Host), cfg.Ratio404) {
		success = false
		//seems to be a soft 404 lol
		return headResp, content, success
	}
	return headResp, content, true
}

func RedirectHandler(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}
