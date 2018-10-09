package librecursebuster

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
)

//ManageRequests handles the request workers
func ManageRequests(cfg *Config, pages, newPages, confirmed chan SpiderPage, workers chan struct{}, printChan chan OutLine, testChan chan string) {
	//manages net request workers
	for {
		page := <-pages
		if gState.Blacklist[page.URL] {
			gState.wg.Done()
			PrintOutput(fmt.Sprintf("Not testing blacklisted URL: %s", page.URL), Info, 0, printChan)
			continue
		}
		for _, method := range gState.Methods {
			if page.Result == nil && !cfg.NoBase {
				workers <- struct{}{}
				gState.wg.Add(1)
				go testURL(cfg, method, page.URL, gState.Client, newPages, workers, confirmed, printChan, testChan)
			}
			if cfg.Wordlist != "" && string(page.URL[len(page.URL)-1]) == "/" { //if we are testing a directory

				//check for wildcard response

				//	maxDirs <- struct{}{}
				dirBust(cfg, page, workers, pages, newPages, confirmed, printChan, testChan)
			}
		}
		gState.wg.Done()

	}
}

//ManageNewURLs will take in any URL, and decide if it should be added to the queue for bustin', or if we discovered something new
func ManageNewURLs(cfg *Config, pages, newpages chan SpiderPage, printChan chan OutLine) {
	//decides on whether to add to the directory list, or add to file output
	for {
		candidate := <-newpages

		//check the candidate is an actual URL
		u, err := url.Parse(strings.TrimSpace(candidate.URL))

		if err != nil {
			gState.wg.Done()
			PrintOutput(err.Error(), Error, 0, printChan)
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
			!cfg.NoRecursion { //no recursion means we don't care about adding extra paths or content
			gState.Checked[actualURL] = true
			gState.CMut.Unlock()
			gState.wg.Add(1)
			pages <- SpiderPage{URL: actualURL, Reference: candidate.Reference, Result: candidate.Result}
			PrintOutput("URL Added: "+actualURL, Debug, 3, printChan)

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
				newpages <- newPage
			}
		} else {
			gState.CMut.Unlock()
		}

		gState.wg.Done()
	}
}

func testURL(cfg *Config, method string, urlString string, client *http.Client,
	newPages chan SpiderPage, workers chan struct{},
	confirmedGood chan SpiderPage, printChan chan OutLine, testChan chan string) {
	defer func() {
		gState.wg.Done()
		atomic.AddUint64(gState.TotalTested, 1)
	}()
	select {
	case testChan <- method + ":" + urlString:
	default: //this is to prevent blocking, it doesn't _really_ matter if it doesn't get written to output
	}
	headResp, content, good := evaluateURL(cfg, method, urlString, client, workers, printChan)

	if !good && !cfg.ShowAll {
		return
	}

	//inception (but also check if the directory version is good, and add that to the queue too)
	if string(urlString[len(urlString)-1]) != "/" && good {
		gState.wg.Add(1)
		newPages <- SpiderPage{URL: urlString + "/", Reference: headResp.Request.URL, Result: headResp}
	}

	gState.wg.Add(1)
	confirmedGood <- SpiderPage{URL: urlString, Result: headResp, Reference: headResp.Request.URL}

	if !cfg.NoSpider && good && !cfg.NoRecursion {
		urls, err := getUrls(content, printChan)
		if err != nil {
			PrintOutput(err.Error(), Error, 0, printChan)
		}
		for _, x := range urls { //add any found pages into the pool
			//add all the directories
			newPage := SpiderPage{}
			newPage.URL = x
			newPage.Reference = headResp.Request.URL

			PrintOutput(
				fmt.Sprintf("Found URL on page: %s", x),
				Debug, 3, printChan,
			)

			gState.wg.Add(1)
			newPages <- newPage
		}
	}
}

func dirBust(cfg *Config, page SpiderPage, workers chan struct{}, pages, newPages, confirmed chan SpiderPage, printChan chan OutLine, testChan chan string) {
	//ugh
	u, err := url.Parse(page.URL)
	if err != nil {
		PrintOutput("This should never occur, url parse error on parsed url?"+err.Error(), Error, 0, printChan)
		return
	}
	//check to make sure we aren't dirbusting a wildcardyboi (NOTE!!! USES FIRST SPECIFIED MEHTOD TO DO SOFT 404!)
	if !cfg.NoWildcardChecks {
		workers <- struct{}{}
		h, _, res := evaluateURL(cfg, gState.Methods[0], page.URL+RandString(printChan), gState.Client, workers, printChan)

		if res { //true response indicates a good response for a guid path, unlikely good
			if detectSoft404(h, gState.Hosts.Get404(u.Host), cfg.Ratio404) {
				//it's a soft404 probably, guess we can continue (this logic seems wrong??)
			} else {
				PrintOutput(
					fmt.Sprintf("Wildcard response detected, skipping dirbusting of %s", page.URL),
					Info, 0, printChan)
				return
			}
		}
	}

	if !cfg.NoStartStop {
		PrintOutput(
			fmt.Sprintf("Dirbusting %s", page.URL),
			Info, 0, printChan,
		)
	}

	atomic.StoreUint32(gState.DirbProgress, 0)

	tested := make(map[string]bool)        //ensure we don't send things more than once
	for _, word := range gState.WordList { //will receive from the channel until it's closed
		//read words off the channel, and test it OR close out because we wanna skip it
		if word == "" {
			atomic.AddUint32(gState.DirbProgress, 1)
			continue
		}
		select {
		case <-gState.StopDir:
			//<-maxDirs
			if !cfg.NoStartStop {
				PrintOutput(fmt.Sprintf("Finished dirbusting: %s", page.URL), Info, 0, printChan)
			}
			return
		default:
			for _, method := range gState.Methods {

				if len(gState.Extensions) > 0 && gState.Extensions[0] != "" {
					for _, ext := range gState.Extensions {
						if tested[page.URL+word+"."+ext] {
							continue
						}
						workers <- struct{}{}
						gState.wg.Add(1)
						go testURL(cfg, method, page.URL+word+"."+ext, gState.Client, newPages, workers, confirmed, printChan, testChan)
						tested[page.URL+word+"."+ext] = true
					}
				}
				if cfg.AppendDir {
					if tested[page.URL+word+"/"] {
						continue
					}
					workers <- struct{}{}
					gState.wg.Add(1)
					go testURL(cfg, method, page.URL+word+"/", gState.Client, newPages, workers, confirmed, printChan, testChan)
					tested[page.URL+word+"/"] = true
				}
				if tested[page.URL+word] {
					continue
				}
				workers <- struct{}{}
				gState.wg.Add(1)
				go testURL(cfg, method, page.URL+word, gState.Client, newPages, workers, confirmed, printChan, testChan)
				tested[page.URL+word] = true
				//if cfg.MaxDirs == 1 {
				atomic.AddUint32(gState.DirbProgress, 1)
				//}
			}
		}
	}
	//<-maxDirs
	if !cfg.NoStartStop {
		PrintOutput(fmt.Sprintf("Finished dirbusting: %s", page.URL), Info, 0, printChan)
	}
}

func StartBusting(cfg *Config, workers chan struct{}, printChan chan OutLine, pages chan SpiderPage, randURL string, u url.URL) {
	defer gState.wg.Done()
	if !cfg.NoWildcardChecks {
		resp, err := HTTPReq("GET", randURL, gState.Client, cfg)
		<-workers
		if err != nil {
			if cfg.InputList != "" {
				PrintOutput(
					err.Error(),
					Error,
					0,
					printChan,
				)
				return
			}
			panic("Canary Error, check url is correct: " + randURL + "\n" + err.Error())

		}
		PrintOutput(
			fmt.Sprintf("Canary sent: %s, Response: %v", randURL, resp.Status),
			Debug, 2, printChan,
		)
		content, _ := ioutil.ReadAll(resp.Body)
		gState.Hosts.AddSoft404Content(u.Host, content, resp) // Soft404ResponseBody = xx
	} else {
		<-workers
	}
	x := SpiderPage{}
	x.URL = u.String()
	x.Reference = &u

	gState.CMut.Lock()
	defer gState.CMut.Unlock()
	if ok := gState.Checked[u.String()+"/"]; !strings.HasSuffix(u.String(), "/") && !ok {
		gState.wg.Add(1)
		pages <- SpiderPage{
			URL:       u.String() + "/",
			Reference: &u,
		}
		gState.Checked[u.String()+"/"] = true
		PrintOutput("URL Added: "+u.String()+"/", Debug, 3, printChan)
	}
	if ok := gState.Checked[x.URL]; !ok {
		gState.wg.Add(1)
		pages <- x
		gState.Checked[x.URL] = true
		PrintOutput("URL Added: "+x.URL, Debug, 3, printChan)
	}
}
