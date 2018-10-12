package librecursebuster

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/c-sto/recursebuster/librecursebuster/testserver"
)

const localURL = "http://localhost:"

/*

200 (OK)
/
/a
/a/b
/a/b/c
/a/
/a/x (200, but same body as /x (404))
/a/y (200, but very similar body to /x (404))

300
/b -> /a/ (302)
/b/c -> /a/b (301)
/b/c/ -> /a/b/c (302)
/b/x (302, but same body as /x (404))
/b/y (301, but very similar body to /x (404))

400
/x (404)
/a/b/c/ (401)
/a/b/c/d (403)

500
/c/d
/c

*/

func TestAppendSlash(t *testing.T) {
	//add an appendslash value to the wordlist that should _only_ be found if the appendslash var is set
	finished := make(chan struct{})
	urlSlice := preSetupTest(nil, "2002", finished)
	gState.Cfg.AppendDir = true
	gState.WordList = append(gState.WordList, "appendslash")
	found := postSetupTest(urlSlice)

	gState.Wait()

	if x, ok := found["/appendslash/"]; !ok || !x {
		panic("didn't find it?")
	}
	close(finished)
}

func TestBasicFunctionality(t *testing.T) {

	finished := make(chan struct{})
	urlSlice := preSetupTest(nil, "2001", finished)
	found := postSetupTest(urlSlice)

	//waitgroup check (if test times out, the waitgroup is broken... somewhere)
	gState.Wait()

	//check for each specific line that should be in there..
	tested := []string{}
	ok200 := []string{
		"/a", "/a/b", "/a/b/c", "/a/",
	}
	for _, i := range ok200 {
		tested = append(tested, i)
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}
	ok300 := []string{
		"/b", "/b/c",
	}
	for _, i := range ok300 {
		tested = append(tested, i)
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}
	ok400 := []string{
		"/a/b/c/", "/a/b/c/d",
	}
	for _, i := range ok400 {
		tested = append(tested, i)
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}
	ok500 := []string{
		"/c/d", "/c", "/c/",
	}
	for _, i := range ok500 {
		tested = append(tested, i)
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}

	//check for values that should not have been found
	for k := range found {
		if strings.Contains(k, "z") {
			panic("Found (but should not have) " + k)
		}
	}
	close(finished)
}

func postSetupTest(urlSlice []string) (found map[string]bool) {
	//start up the management goroutines
	go ManageRequests()
	go ManageNewURLs()

	//default turn url into a url object call
	u, err := url.Parse(urlSlice[0])
	if err != nil {
		panic(err)
	}

	//canary things
	prefix := u.String()
	if len(prefix) > 0 && string(prefix[len(prefix)-1]) != "/" {
		prefix = prefix + "/"
	}
	randURL := fmt.Sprintf("%s%s", prefix, gState.Cfg.Canary)
	gState.AddWG()
	gState.Chans.GetWorkers() <- struct{}{}
	go StartBusting(randURL, *u)

	//start the print channel (so that we can see output if a test fails)
	go func() {
		for {
			p := <-gState.Chans.printChan
			gState.wg.Done()
			if p.Content != "" {

			}
			//fmt.Println(p.Content)
		}
	}()

	//use the found map to determine later on if we have found the expected URL's
	found = make(map[string]bool)
	go func() {
		t := time.NewTicker(1 * time.Second).C
		for {
			select {
			case x := <-gState.Chans.confirmedChan:
				gState.wg.Done()
				u, e := url.Parse(x.URL)
				if e != nil {
					panic(e)
				}
				found[u.Path] = true
				//fmt.Println("CONFIRMED!", x)
			case <-t:
				//fmt.Println(globalState.wg)
			}
		}
	}()
	return
}

func preSetupTest(cfg *Config, servPort string, finished chan struct{}) (urlSlice []string) {
	//Test default functions. Basic dirb should work, and all files should be found as expected

	//basic state setup
	globalState := State{}.Init()
	globalState.Hosts.Init()

	//start the test server
	setup := make(chan struct{})
	go testserver.Start(servPort, finished, setup)
	<-setup //whoa, concurrency sucks???
	//test URL
	globalState.Cfg.URL = localURL + servPort

	//default slice starter-upper
	urlSlice = getURLSlice(globalState)

	//setup the config to default values
	setupConfig(globalState, urlSlice[0], cfg)

	//normal main setup state call
	SetupState(globalState)

	//test wordlist to use
	wordlist := `a
b
c
d
e
x
y
z
`
	//overwrite the 'no wordlist' setup for the state
	globalState.WordList = strings.Split(wordlist, "\n")
	globalState.Cfg.Wordlist = "test"
	globalState.DirbProgress = new(uint32)

	return
}

func setupConfig(globalState *State, urlSliceZero string, cfg *Config) {
	globalState.Cfg.Version = "TEST"
	totesTested := uint64(0)
	globalState.TotalTested = &totesTested
	globalState.Cfg.ShowAll = false
	globalState.Cfg.AppendDir = true
	globalState.Cfg.Auth = ""
	globalState.Cfg.BadResponses = "404"
	//globalState.Cfg.BadHeader, "Check for presence of this header. If an exact match is found"
	//globalState.Cfg.BodyContent, ""
	globalState.Cfg.BlacklistLocation = ""
	globalState.Cfg.Canary = ""
	globalState.Cfg.CleanOutput = false
	globalState.Cfg.Cookies = ""
	globalState.Cfg.Debug = false
	//globalState.Cfg.MaxDirs = 1
	globalState.Cfg.Extensions = ""
	//globalState.Cfg.Headers = "Additional headers to include with request. Supply as key:value. Can specify multiple - eg '-headers X-Forwarded-For:127.0.01 -headers X-ATT-DeviceId:XXXXX'")
	globalState.Cfg.HTTPS = false
	globalState.Cfg.InputList = ""
	globalState.Cfg.SSLIgnore = false
	globalState.Cfg.ShowLen = false
	globalState.Cfg.NoBase = false
	globalState.Cfg.NoGet = false
	globalState.Cfg.NoHead = false
	globalState.Cfg.NoRecursion = false
	globalState.Cfg.NoSpider = false
	globalState.Cfg.NoStatus = false
	globalState.Cfg.NoStartStop = false
	globalState.Cfg.NoWildcardChecks = false
	globalState.Cfg.NoUI = true
	globalState.Cfg.Localpath = "." + string(os.PathSeparator) + "busted.txt"
	globalState.Cfg.Methods = "GET"
	globalState.Cfg.ProxyAddr = ""
	globalState.Cfg.Ratio404 = 0.95
	globalState.Cfg.FollowRedirects = false
	globalState.Cfg.BurpMode = false
	globalState.Cfg.Threads = 1
	globalState.Cfg.Timeout = 20
	globalState.Cfg.URL = ""
	globalState.Cfg.Agent = "RecurseBuster/" + globalState.Cfg.Version
	globalState.Cfg.VerboseLevel = 0
	globalState.Cfg.ShowVersion = false
	globalState.Cfg.Wordlist = ""
	globalState.Cfg.WhitelistLocation = ""

	globalState.Cfg.URL = localURL

	if cfg != nil {
		globalState.Cfg = cfg
	}

	var h *url.URL
	var err error
	h, err = url.Parse(urlSliceZero)
	if err != nil {
		panic(err)
	}

	if h.Scheme == "" {
		if globalState.Cfg.HTTPS {
			h, err = url.Parse("https://" + urlSliceZero)
		} else {
			h, err = url.Parse("http://" + urlSliceZero)
		}
	}
	if err != nil {
		panic(err)
	}
	globalState.Hosts.AddHost(h)

	if globalState.Cfg.Canary == "" {
		globalState.Cfg.Canary = RandString()
	}
}
