package librecursebuster

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

func HttpReq(method, path string, client *http.Client, cfg Config) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, path, nil)
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

func evaluateURL(wg *sync.WaitGroup, cfg Config, state State, urlString string, client *http.Client, workers chan struct{}, printChan chan OutLine) (headResp *http.Response, content []byte, success bool) {
	success = true
	headResp, _, err := HttpReq("HEAD", urlString, client, cfg) //send a HEAD. Ignore body response
	if err != nil {
		success = false
		<-workers //done with the net thread
		PrintOutput(fmt.Sprintf("%s", err), Error, 0, wg, printChan)
		return
	}

	//Check if we care about it (header only) section
	if state.BadResponses[headResp.StatusCode] {
		success = false
		<-workers
		return
	}

	//this is all we have to do if we aren't doing GET's
	if cfg.NoGet {
		<-workers
		return
	}

	//get content from validated path/file thing
	headResp, content, err = HttpReq("GET", urlString, client, cfg)
	<-workers //done with the net thread
	if err != nil {
		success = false
		PrintOutput(fmt.Sprintf("%s", err), Error, 0, wg, printChan)

		return //probably handle better
	}

	//check we care about it (body only) section
	//double check that it's not 404/error using smart blockchain AI tech
	PrintOutput(
		fmt.Sprintf("%v, %v, %v",
			content, state.Soft404ResponseBody,
			detectSoft404(content, state.Soft404ResponseBody, cfg.Ratio404)),
		Debug, 4, wg, printChan)
	if detectSoft404(content, state.Soft404ResponseBody, cfg.Ratio404) {
		success = false
		//seems to be a soft 404 lol
		return
	}
	return
}

func RedirectHandler(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}
