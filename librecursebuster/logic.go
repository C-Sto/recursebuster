package librecursebuster

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

//ManageRequests handles the request workers
func ManageRequests(cfg Config, state State, wg *sync.WaitGroup, pages, newPages, confirmed chan SpiderPage, workers chan struct{}, printChan chan OutLine, maxDirs chan struct{}, testChan chan string) {
	//manages net request workers
	for {
		page := <-pages
		if state.Blacklist[page.URL] {
			wg.Done()
			PrintOutput(fmt.Sprintf("Not testing blacklisted URL: %s", page.URL), Info, 0, wg, printChan)
			continue
		}
		for _, method := range state.Methods {
			if !cfg.NoBase {
				workers <- struct{}{}
				wg.Add(1)
				go testURL(cfg, state, wg, method, page.URL, state.Client, newPages, workers, confirmed, printChan, testChan)
			}
			if cfg.Wordlist != "" && string(page.URL[len(page.URL)-1]) == "/" { //if we are testing a directory

				//check for wildcard response

				wg.Add(1)
				go dirBust(cfg, state, page, wg, workers, pages, newPages, confirmed, printChan, maxDirs, testChan)
			}
		}
		wg.Done()
	}
}

//ManageNewURLs will take in any URL, and decide if it should be added to the queue for bustin', or if we discovered something new
func ManageNewURLs(cfg Config, state State, wg *sync.WaitGroup, pages, newpages chan SpiderPage, printChan chan OutLine) {
	//decides on whether to add to the directory list, or add to file output
	checked := make(map[string]bool)
	//	preCheck := make(map[string]bool)
	for {
		candidate := <-newpages

		/*//shortcut (will make checked much bigger than it should be, but will save cycles)
		//removed due to stupid memory soaking issue
		if _, ok := preCheck[candidate.URL]; ok {
			wg.Done()
			continue
		}
		preCheck[candidate.URL] = true
		*/

		//check the candidate is an actual URL
		u, err := url.Parse(strings.TrimSpace(candidate.URL))

		if err != nil {
			wg.Done()
			PrintOutput(err.Error(), Error, 0, wg, printChan)
			continue //probably a better way of doing this
		}

		//links of the form <a href="/thing" ></a> don't have a host portion to the URL
		if len(u.Host) == 0 {
			u.Host = candidate.Reference.Host
		}

		//actualUrl := state.ParsedURL.Scheme + "://" + u.Host
		actualURL := cleanURL(u, (*candidate.Reference).Scheme+"://"+u.Host)

		if _, ok := checked[actualURL]; !ok && //must have not checked it before
			(state.Hosts.HostExists(u.Host) || state.Whitelist[u.Host]) && //must be within whitelist, or be one of the starting urls
			!cfg.NoRecursion { //no recursion means we don't care about adding extra paths or content

			checked[actualURL] = true

			wg.Add(1)
			pages <- SpiderPage{URL: actualURL, Reference: candidate.Reference}

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
				wg.Add(1)
				newpages <- newPage
			}
		}

		wg.Done()
	}
}

func testURL(cfg Config, state State, wg *sync.WaitGroup, method string, urlString string, client *http.Client,
	newPages chan SpiderPage, workers chan struct{},
	confirmedGood chan SpiderPage, printChan chan OutLine, testChan chan string) {
	defer func() {
		wg.Done()
		atomic.AddUint64(state.TotalTested, 1)
	}()

	select {
	case testChan <- method + ":" + urlString:
	default: //this is to prevent blocking, it doesn't _really_ matter if it doesn't get written to output
	}

	headResp, content, good := evaluateURL(wg, cfg, state, method, urlString, client, workers, printChan)

	if !good && !cfg.ShowAll {
		return
	}

	//inception (but also check if the directory version is good, and add that to the queue too)
	if string(urlString[len(urlString)-1]) != "/" && good {
		wg.Add(1)
		newPages <- SpiderPage{URL: urlString + "/", Reference: headResp.Request.URL}
	}

	wg.Add(1)
	confirmedGood <- SpiderPage{URL: urlString, Result: headResp, Reference: headResp.Request.URL}

	if !cfg.NoSpider && good && !cfg.NoRecursion {
		urls, err := getUrls(content, printChan)
		if err != nil {
			PrintOutput(err.Error(), Error, 0, wg, printChan)
		}
		for _, x := range urls { //add any found pages into the pool
			//add all the directories
			newPage := SpiderPage{}
			newPage.URL = x
			newPage.Reference = headResp.Request.URL

			PrintOutput(
				fmt.Sprintf("Found URL on page: %s", x),
				Debug, 3, wg, printChan,
			)

			wg.Add(1)
			newPages <- newPage
		}
	}
}

func dirBust(cfg Config, state State, page SpiderPage, wg *sync.WaitGroup, workers chan struct{}, pages, newPages, confirmed chan SpiderPage, printChan chan OutLine, maxDirs chan struct{}, testChan chan string) {
	defer wg.Done()

	//ugh
	u, err := url.Parse(page.URL)
	if err != nil {
		PrintOutput("This should never occur, url parse error on parsed url?"+err.Error(), Error, 0, wg, printChan)
		return
	}
	//check to make sure we aren't dirbusting a wildcardyboi (NOTE!!! USES FIRST SPECIFIED MEHTOD TO DO SOFT 404!)
	if !cfg.NoWildcardChecks {
		workers <- struct{}{}
		_, x, res := evaluateURL(wg, cfg, state, state.Methods[0], page.URL+RandString(printChan), state.Client, workers, printChan)

		if res { //true response indicates a good response for a guid path, unlikely good
			if detectSoft404(x, state.Hosts.Get404Body(u.Host), cfg.Ratio404) {
				//it's a soft404 probably, guess we can continue (this logic seems wrong??)
			} else {
				PrintOutput(
					fmt.Sprintf("Wildcard response detected, skipping dirbusting of %s", page.URL),
					Info, 0, wg, printChan)
				return
			}
		}
	}
	maxDirs <- struct{}{}

	//load in the wordlist to a channel (can probs be async)
	wordsChan := make(chan string, 300) //don't expect we will need it much bigger than this

	go LoadWords(cfg.Wordlist, wordsChan, printChan) //wordlist management doesn't need waitgroups, because of the following range statement
	if !cfg.NoStartStop {
		PrintOutput(
			fmt.Sprintf("Dirbusting %s", page.URL),
			Info, 0, wg, printChan,
		)
	}
	if cfg.MaxDirs == 1 {
		atomic.StoreUint32(state.DirbProgress, 0)
	}
	for word := range wordsChan { //will receive from the channel until it's closed
		//read words off the channel, and test it
		for _, method := range state.Methods {

			if len(state.Extensions) > 0 && state.Extensions[0] != "" {
				for _, ext := range state.Extensions {
					workers <- struct{}{}
					wg.Add(1)
					go testURL(cfg, state, wg, method, page.URL+word+"."+ext, state.Client, newPages, workers, confirmed, printChan, testChan)
				}
			}
			if cfg.AppendDir {
				workers <- struct{}{}
				wg.Add(1)
				go testURL(cfg, state, wg, method, page.URL+word+"/", state.Client, newPages, workers, confirmed, printChan, testChan)
			}
			workers <- struct{}{}
			wg.Add(1)
			go testURL(cfg, state, wg, method, page.URL+word, state.Client, newPages, workers, confirmed, printChan, testChan)

			if cfg.MaxDirs == 1 {
				atomic.AddUint32(state.DirbProgress, 1)
			}
		}
	}
	<-maxDirs
	if !cfg.NoStartStop {
		PrintOutput(fmt.Sprintf("Finished dirbusting: %s", page.URL), Info, 0, wg, printChan)
	}
}
