package librecursebuster

import (
	"bytes"
	"fmt"
	"net/http"
)

func HttpReq(method, path string, client *http.Client, cfg Config) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, path, nil)
	req.Header.Set("User-Agent", cfg.Agent)
	if cfg.Cookies != "" {
		req.Header.Set("Cookie", cfg.Cookies)
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

func evaluateURL(cfg Config, state State, urlString string, client *http.Client, workers chan struct{}, printChan chan OutLine) (headResp *http.Response, content []byte, success bool) {
	success = true
	headResp, _, err := HttpReq("HEAD", urlString, client, cfg) //send a HEAD. Ignore body response
	if err != nil {
		success = false
		<-workers //done with the net thread
		printChan <- OutLine{
			Content: fmt.Sprintf("%s", err),
			Type:    Error,
		}
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
		printChan <- OutLine{
			Content: fmt.Sprintf("%s", err),
			Type:    Error,
		}
		return //probably handl/n
	}

	//check we care about it (body only) section
	//double check that it's not 404/error using smart blockchain AI tech
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
