package recursebuster

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/c-sto/recursebuster/pkg/net"
)

func (gState *State) evaluateURL(method string, urlString string, client *http.Client) (headResp *http.Response, content []byte, success bool) {

	//optimize GET requests by sending a head first (it's cheaper)
	if method == "GET" && !gState.Cfg.NoHead {
		headResp, err := gState.requester.HTTPReq("HEAD", urlString, client) //send a HEAD. Ignore body response
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
				gState.requester.HTTPReq("HEAD", urlString, gState.BurpClient) //send a HEAD. Ignore body response
			}
			//<-gState.Chans.workersChan
			return headResp, content, success
		}
	}

	headResp, err := gState.requester.HTTPReq(method, urlString, client)
	if err != nil {
		gState.PrintOutput(fmt.Sprintf("%s", err), Error, 0)
		return headResp, content, false
	}
	content, err = ioutil.ReadAll(headResp.Body)

	if gState.Cfg.BadBod != "" {
		if bytes.Contains(content, []byte(gState.Cfg.BadBod)) {
			return headResp, content, false
		}
	}
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
	is404, _ := net.DetectSoft404(headResp, gState.Hosts.Get404(headResp.Request.Host), gState.Cfg.Ratio404)

	if is404 {
		//seems to be a soft 404 lol
		return headResp, content, false
	}

	if gState.Cfg.BurpMode {
		gState.requester.HTTPReq(method, urlString, gState.BurpClient)
	}
	return headResp, content, true
}
