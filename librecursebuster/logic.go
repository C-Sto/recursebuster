package librecursebuster

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
)

//ManageRequests handles the request workers
func (gState *State) ManageRequests() {
	//manages net request workers
	for {
		page := <-gState.Chans.pagesChan
		if gState.Blacklist[page.URL] {
			gState.wg.Done()
			gState.PrintOutput(fmt.Sprintf("Not testing blacklisted URL: %s", page.URL), Info, 0)
			continue
		}
		for _, method := range gState.Methods {
			if page.Result == nil && !gState.Cfg.NoBase {
				gState.Chans.workersChan <- struct{}{}
				gState.wg.Add(1)
				go gState.testURL(method, page.URL, gState.Client)
			}
			if gState.Cfg.Wordlist != "" && string(page.URL[len(page.URL)-1]) == "/" { //if we are testing a directory

				//check for wildcard response

				//	maxDirs <- struct{}{}
				gState.dirBust(page)
			}
		}
		gState.wg.Done()

	}
}

//ManageNewURLs will take in any URL, and decide if it should be added to the queue for bustin', or if we discovered something new
func (gState *State) ManageNewURLs() {
	//decides on whether to add to the directory list, or add to file output
	for {
		candidate, more := <-gState.Chans.newPagesChan
		if !more {
			return
		}
		//check the candidate is an actual URL
		u, err := url.Parse(strings.TrimSpace(candidate.URL))

		if err != nil {
			gState.wg.Done()
			gState.PrintOutput(err.Error(), Error, 0)
			continue //probably a better way of doing this
		}

		//links of the form <a href="/thing" ></a> don't have a host portion to the URL
		if len(u.Host) == 0 {
			u.Host = candidate.Reference.Host
		}

		//actualUrl := gState.ParsedURL.Scheme + "://" + u.Host
		actualURL := cleanURL(u, (*candidate.Reference).Scheme+"://"+u.Host)

		gState.CMut.Lock()
		if _, ok := gState.Checked[actualURL]; !ok && //must have not checked it before
			(gState.Hosts.HostExists(u.Host) || gState.Whitelist[u.Host]) && //must be within whitelist, or be one of the starting urls
			!gState.Cfg.NoRecursion { //no recursion means we don't care about adding extra paths or content
			gState.Checked[actualURL] = true
			gState.CMut.Unlock()
			gState.wg.Add(1)
			gState.Chans.pagesChan <- SpiderPage{URL: actualURL, Reference: candidate.Reference, Result: candidate.Result}
			gState.PrintOutput("URL Added: "+actualURL, Debug, 3)

			//also add any directories in the supplied path to the 'to be hacked' queue
			path := ""
			dirs := strings.Split(u.Path, "/")
			for i, y := range dirs {

				path = path + y
				if len(path) > 0 && string(path[len(path)-1]) != "/" && i != len(dirs)-1 {
					path = path + "/" //don't add double /'s, and don't add on the last value
				}
				//prepend / if it doesn't already exist
				if len(path) > 0 && string(path[0]) != "/" {
					path = "/" + path
				}

				newDir := candidate.Reference.Scheme + "://" + candidate.Reference.Host + path
				newPage := SpiderPage{}
				newPage.URL = newDir
				newPage.Reference = candidate.Reference
				newPage.Result = candidate.Result
				gState.CMut.RLock()
				if gState.Checked[newDir] {
					gState.CMut.RUnlock()
					continue
				}
				gState.CMut.RUnlock()
				gState.wg.Add(1)
				gState.Chans.newPagesChan <- newPage
			}
		} else {
			gState.CMut.Unlock()
		}

		gState.wg.Done()
	}
}

func (gState *State) testURL(method string, urlString string, client *http.Client) {
	defer func() {
		gState.wg.Done()
		atomic.AddUint64(gState.TotalTested, 1)
	}()
	select {
	case gState.Chans.testChan <- method + ":" + urlString:
	default: //this is to prevent blocking, it doesn't _really_ matter if it doesn't get written to output
	}
	headResp, content, good := gState.evaluateURL(method, urlString, client)

	if !good && !gState.Cfg.ShowAll {
		return
	}

	//inception (but also check if the directory version is good, and add that to the queue too)
	if string(urlString[len(urlString)-1]) != "/" && good {
		gState.wg.Add(1)
		gState.Chans.newPagesChan <- SpiderPage{URL: urlString + "/", Reference: headResp.Request.URL, Result: headResp}
	}

	gState.wg.Add(1)
	if headResp == nil {
		gState.Chans.confirmedChan <- SpiderPage{URL: urlString, Result: headResp, Reference: nil}
	} else {
		gState.Chans.confirmedChan <- SpiderPage{URL: urlString, Result: headResp, Reference: headResp.Request.URL}
	}
	if !gState.Cfg.NoSpider && good && !gState.Cfg.NoRecursion {
		urls, err := getUrls(content)
		if err != nil {
			gState.PrintOutput(err.Error(), Error, 0)
		}
		for _, x := range urls { //add any found pages into the pool
			//add all the directories
			newPage := SpiderPage{}
			newPage.URL = x
			newPage.Reference = headResp.Request.URL

			gState.PrintOutput(
				fmt.Sprintf("Found URL on page: %s", x),
				Debug, 3,
			)

			gState.wg.Add(1)
			gState.Chans.newPagesChan <- newPage
		}
	}
}

func (gState *State) dirBust(page SpiderPage) {
	//ugh
	u, err := url.Parse(page.URL)
	if err != nil {
		gState.PrintOutput("This should never occur, url parse error on parsed url?"+err.Error(), Error, 0)
		return
	}
	//check to make sure we aren't dirbusting a wildcardyboi (NOTE!!! USES FIRST SPECIFIED MEHTOD TO DO SOFT 404!)
	if !gState.Cfg.NoWildcardChecks {
		gState.Chans.workersChan <- struct{}{}
		h, _, res := gState.evaluateURL(gState.Methods[0], page.URL+RandString(), gState.Client)
		//fmt.Println(page.URL, h, res)
		if res { //true response indicates a good response for a guid path, unlikely good
			is404, _ := detectSoft404(h, gState.Hosts.Get404(u.Host), gState.Cfg.Ratio404)
			if is404 {
				//it's a soft404 probably, guess we can continue (this logic seems wrong??)
			} else {
				gState.PrintOutput(
					fmt.Sprintf("Wildcard response detected, skipping dirbusting of %s", page.URL),
					Info, 0)
				return
			}
		}
	}

	if !gState.Cfg.NoStartStop {
		gState.PrintOutput(
			fmt.Sprintf("Dirbusting %s", page.URL),
			Info, 0,
		)
	}

	atomic.StoreUint32(gState.DirbProgress, 0)
	//ensure we don't send things more than once
	for _, word := range gState.WordList { //will receive from the channel until it's closed
		//read words off the channel, and test it OR close out because we wanna skip it
		if word == "" {
			atomic.AddUint32(gState.DirbProgress, 1)
			continue
		}
		select {
		case <-gState.StopDir:
			//<-maxDirs
			if !gState.Cfg.NoStartStop {
				gState.PrintOutput(fmt.Sprintf("Finished dirbusting: %s", page.URL), Info, 0)
			}
			return
		default:
			for _, method := range gState.Methods {
				if len(gState.Extensions) > 0 && gState.Extensions[0] != "" {
					for _, ext := range gState.Extensions {
						gState.CMut.Lock()
						if gState.Checked[method+page.URL+word+"."+ext] {
							gState.CMut.Unlock()
							continue
						}
						gState.Chans.workersChan <- struct{}{}
						gState.wg.Add(1)
						go gState.testURL(method, page.URL+word+"."+ext, gState.Client)
						gState.Checked[method+page.URL+word+"."+ext] = true
						gState.CMut.Unlock()
					}
				}
				if gState.Cfg.AppendDir {
					gState.CMut.Lock()
					if gState.Checked[method+page.URL+word+"/"] {
						gState.CMut.Unlock()
						continue
					}
					gState.Chans.workersChan <- struct{}{}
					gState.wg.Add(1)
					go gState.testURL(method, page.URL+word+"/", gState.Client)
					gState.Checked[method+page.URL+word+"/"] = true
					gState.CMut.Unlock()
				}
				gState.CMut.Lock()
				if gState.Checked[method+page.URL+word] {
					gState.CMut.Unlock()
					continue
				}
				gState.Chans.workersChan <- struct{}{}
				gState.wg.Add(1)
				go gState.testURL(method, page.URL+word, gState.Client)
				gState.Checked[method+page.URL+word] = true
				gState.CMut.Unlock()
				//if gState.Cfg.MaxDirs == 1 {
				atomic.AddUint32(gState.DirbProgress, 1)
				//}
			}
		}
	}
	//<-maxDirs
	if !gState.Cfg.NoStartStop {
		gState.PrintOutput(fmt.Sprintf("Finished dirbusting: %s", page.URL), Info, 0)
	}
}

//StartBusting will add a suppllied url to the queue to be tested
func (gState *State) StartBusting(randURL string, u url.URL) {
	defer gState.wg.Done()
	if !gState.Cfg.NoWildcardChecks {
		resp, err := gState.HTTPReq("GET", randURL, gState.Client)
		<-gState.Chans.workersChan
		if err != nil {
			if gState.Cfg.InputList != "" {
				gState.PrintOutput(
					err.Error(),
					Error,
					0,
				)
				return
			}
			panic("Canary Error, check url is correct: " + randURL + "\n" + err.Error())

		}
		gState.PrintOutput(
			fmt.Sprintf("Canary sent: %s, Response: %v", randURL, resp.Status),
			Debug, 2,
		)
		content, _ := ioutil.ReadAll(resp.Body)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(content))
		gState.Hosts.AddSoft404Content(u.Host, content, resp) // Soft404ResponseBody = xx
	} else {
		<-gState.Chans.workersChan
	}
	x := SpiderPage{}
	x.URL = u.String()
	x.Reference = &u

	gState.CMut.Lock()
	defer gState.CMut.Unlock()
	if ok := gState.Checked[u.String()+"/"]; !strings.HasSuffix(u.String(), "/") && !ok {
		gState.wg.Add(1)
		gState.Chans.pagesChan <- SpiderPage{
			URL:       u.String() + "/",
			Reference: &u,
		}
		gState.Checked[u.String()+"/"] = true
		gState.PrintOutput("URL Added: "+u.String()+"/", Debug, 3)
	}
	if ok := gState.Checked[x.URL]; !ok {
		gState.wg.Add(1)
		gState.Chans.pagesChan <- x
		gState.Checked[x.URL] = true
		gState.PrintOutput("URL Added: "+x.URL, Debug, 3)
	}
}
