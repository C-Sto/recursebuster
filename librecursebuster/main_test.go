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

func TestBasicFunctionality(t *testing.T) {

	finished := make(chan struct{})
	cfg := getDefaultConfig()
	urlSlice := preSetupTest(cfg, "2001", finished, t)
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

func TestAppendSlash(t *testing.T) {
	//add an appendslash value to the wordlist that should _only_ be found if the appendslash var is set
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	urlSlice := preSetupTest(cfg, "2002", finished, t)
	gState.Cfg.AppendDir = true
	gState.WordList = append(gState.WordList, "appendslash")
	found := postSetupTest(urlSlice)
	gState.Wait()

	if x, ok := found["/appendslash/"]; !ok || !x {
		panic("didn't find it?")
	}
	close(finished)
}

func TestBasicAuth(t *testing.T) {
	//ensure that basic auth checks are found
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Auth = "dGVzdDp0ZXN0"
	urlSlice := preSetupTest(cfg, "2003", finished, t)
	gState.WordList = append(gState.WordList, "basicauth")
	found := postSetupTest(urlSlice)
	gState.Wait()

	if x, ok := found["/a/b/c/basicauth"]; !ok || !x {
		panic("Failed basic auth test!")
	}

}

func TestBadCodes(t *testing.T) {
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.BadResponses = "404,500"
	urlSlice := preSetupTest(cfg, "2004", finished, t)
	gState.WordList = append(gState.WordList, "badcode")
	found := postSetupTest(urlSlice)
	gState.Wait()

	for x := range found {
		if strings.Contains(x, "badcode") {
			panic("Failed bad header code test")
		}
	}

}

func TestBadHeaders(t *testing.T) {
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.BadHeader = ArrayStringFlag{}
	cfg.BadHeader.Set("X-Bad-Header: test123")
	urlSlice := preSetupTest(cfg, "2005", finished, t)
	gState.WordList = append(gState.WordList, "badheader")
	found := postSetupTest(urlSlice)
	gState.Wait()

	for x := range found {
		if strings.Contains(x, "badheader") {
			panic("Failed bad header code test")
		}
	}

}

func TestAjax(t *testing.T) {
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Ajax = true
	cfg.Methods = "GET,POST"
	urlSlice := preSetupTest(cfg, "2006", finished, t)
	gState.WordList = append(gState.WordList, "ajaxonly")
	gState.WordList = append(gState.WordList, "onlynoajax")
	gState.WordList = append(gState.WordList, "ajaxpost")
	found := postSetupTest(urlSlice)
	gState.Wait()

	if x, ok := found["/ajaxonly"]; !ok || !x {
		panic("Failed ajax header check")
	}

	if x, ok := found["/ajaxpost"]; !ok || !x {
		panic("Failed ajax header check")
	}

	if x, ok := found["/onlynoajax"]; ok || x {
		panic("Failed ajax header check")
	}

}

func TestBodyContent(t *testing.T) {
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Methods = "GET,POST"
	urlSlice := preSetupTest(cfg, "2007", finished, t)
	gState.WordList = append(gState.WordList, "postbody")
	gState.bodyContent = "test=bodycontent"
	gState.Cfg.BodyContent = "test"
	found := postSetupTest(urlSlice)
	gState.Wait()

	if x, ok := found["/postbody"]; !ok || !x {
		panic("Failed body based request")
	}
}

func TestBlacklist(t *testing.T) {
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	urlSlice := preSetupTest(cfg, "2008", finished, t)
	gState.Cfg.BlacklistLocation = "test"
	gState.Blacklist = make(map[string]bool)
	gState.Blacklist["http://localhost:2008/a/b"] = true
	found := postSetupTest(urlSlice)
	gState.Wait()

	if x, ok := found["/a/b"]; ok || x {
		panic("Failed blacklist testing1")
	}

	if x, ok := found["/a/b/c"]; ok || x {
		panic("Failed blacklist testing2")
	}
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

func preSetupTest(cfg *Config, servPort string, finished chan struct{}, t *testing.T) (urlSlice []string) {
	//Test default functions. Basic dirb should work, and all files should be found as expected

	//basic state setup
	globalState := State{}.Init()
	if cfg != nil {
		globalState.Cfg = cfg
	}
	globalState.Hosts.Init()

	//start the test server
	setup := make(chan struct{})
	go testserver.Start(servPort, finished, setup, t)
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

func getDefaultConfig() *Config {
	return &Config{
		Version:      "TEST",
		ShowAll:      false,
		AppendDir:    true,
		Auth:         "",
		BadResponses: "404",
		BadHeader:    nil, //ArrayStringFlag{} // "" // "Check for presence of this header. If an exact match is found"
		//BodyContent, ""
		BlacklistLocation: "",
		Canary:            "",
		CleanOutput:       false,
		Cookies:           "",
		Debug:             false,
		//MaxDirs: 1
		Extensions:        "",
		Headers:           nil, // "Additional headers to include with request. Supply as key:value. Can specify multiple - eg '-headers X-Forwarded-For:127.0.01 -headers X-ATT-DeviceId:XXXXX'")
		HTTPS:             false,
		InputList:         "",
		SSLIgnore:         false,
		ShowLen:           false,
		NoBase:            false,
		NoGet:             false,
		NoHead:            false,
		NoRecursion:       false,
		NoSpider:          false,
		NoStatus:          false,
		NoStartStop:       false,
		NoWildcardChecks:  false,
		NoUI:              true,
		Localpath:         "." + string(os.PathSeparator) + "busted.txt",
		Methods:           "GET",
		ProxyAddr:         "",
		Ratio404:          0.95,
		FollowRedirects:   false,
		BurpMode:          false,
		Threads:           1,
		Timeout:           20,
		Agent:             "RecurseBuster/" + "TESTING",
		VerboseLevel:      0,
		ShowVersion:       false,
		Wordlist:          "",
		WhitelistLocation: "",

		URL: localURL,
	}

}

func setupConfig(globalState *State, urlSliceZero string, cfg *Config) {

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
