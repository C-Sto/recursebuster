package test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/c-sto/recursebuster/cmd"

	"github.com/c-sto/recursebuster/pkg/recursebuster"

	"github.com/c-sto/recursebuster/test/testserver"
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
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	gState, urlSlice := preSetupTest(cfg, "2001", finished, t)
	found := postSetupTest(urlSlice, gState)

	//waitgroup check (if test times out, the waitgroup is broken... somewhere)
	gState.Wait()

	//check for each specific line that should be in there..
	tested := []string{}
	ok := []string{
		"/a", "/a/b", "/a/b/c", "/a/", "/spideronly",
		"/b", "/b/c",
		"/a/b/c/", "/a/b/c/d",
		"/c/d", "/c", "/c/",
	}
	for _, i := range ok {
		tested = append(tested, i)
		if x, ok := found[i]; !ok || x == nil {
			t.Error("Did not find " + i)
		}
	}

	//check for values that should not have been found
	for k := range found {
		if strings.Contains(k, "z") {
			t.Error("Found (but should not have) " + k)
		}
	}

	if x, ok := found["/a/x"]; ok && x != nil {
		t.Error("Found (but should not have) /a/x")
	}

	if x, ok := found["/a/y"]; ok && x != nil {
		t.Error("Found (but should not have) /a/x")
	}

	close(finished)
}

func TestAppendSlash(t *testing.T) {
	//add an appendslash value to the wordlist that should _only_ be found if the appendslash var is set
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	gState, urlSlice := preSetupTest(cfg, "2002", finished, t)
	gState.Cfg.AppendDir = true
	gState.WordList = append(gState.WordList, "appendslash")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/appendslash/"]; !ok || x == nil {
		t.Error("didn't find it?")
	}
	close(finished)
}

func TestBasicAuth(t *testing.T) {
	t.Parallel()
	//ensure that basic auth checks are found
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Auth = "dGVzdDp0ZXN0"
	gState, urlSlice := preSetupTest(cfg, "2003", finished, t)
	gState.WordList = append(gState.WordList, "basicauth")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/a/b/c/basicauth"]; !ok || x == nil {
		t.Error("Failed basic auth test!")
	}

}

func TestBadCodes(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.BadResponses = "404,500"
	gState, urlSlice := preSetupTest(cfg, "2004", finished, t)
	gState.WordList = append(gState.WordList, "badcode")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	for x := range found {
		if strings.Contains(x, "badcode") {
			t.Error("Failed bad header code test")
		}
	}

}

func TestBadHeaders(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.BadHeader = recursebuster.ArrayStringFlag{}
	cfg.BadHeader.Set("X-Bad-Header: test123")
	gState, urlSlice := preSetupTest(cfg, "2005", finished, t)
	gState.WordList = append(gState.WordList, "badheader")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	for x := range found {
		if strings.Contains(x, "badheader") {
			t.Error("Failed bad header code test")
		}
	}

}

func TestAjax(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Ajax = true
	cfg.Methods = "GET,POST"
	gState, urlSlice := preSetupTest(cfg, "2006", finished, t)
	gState.WordList = append(gState.WordList, "ajaxonly")
	gState.WordList = append(gState.WordList, "onlynoajax")
	gState.WordList = append(gState.WordList, "ajaxpost")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/ajaxonly"]; !ok || x == nil {
		t.Error("Failed ajax header check 1")
	}

	if x, ok := found["/ajaxpost"]; !ok || x == nil {
		t.Error("Failed ajax header check 2")
	}

	if x, ok := found["/onlynoajax"]; ok && x != nil {
		t.Error("Failed ajax header check 3")
	}

}

func TestBodyContent(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Methods = "GET,POST"
	cfg.NoHead = true
	gState, urlSlice := preSetupTest(cfg, "2007", finished, t)
	gState.WordList = append(gState.WordList, "postbody")
	gState.BodyContent = "test=bodycontent"
	gState.Cfg.BodyContent = "test"
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/postbody"]; !ok || x == nil {
		t.Error("Failed body based request")
	}
}

func TestBlacklist(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	gState, urlSlice := preSetupTest(cfg, "2008", finished, t)
	gState.Cfg.BlacklistLocation = "test"
	gState.Blacklist = make(map[string]bool)
	gState.Blacklist["http://localhost:2008/a/b"] = true
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/a/b"]; ok && x != nil {
		t.Error("Failed blacklist testing1")
	}

	if x, ok := found["/a/b/c"]; ok && x != nil {
		t.Error("Failed blacklist testing2")
	}
}

func TestCookies(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Cookies = "lol=ok; cookie2=test;"
	gState, urlSlice := preSetupTest(cfg, "2009", finished, t)
	gState.WordList = append(gState.WordList, "cookiesonly")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/cookiesonly"]; !ok || x == nil {
		t.Error("Failed Cookie test")
	}
}
func TestExt(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Extensions = "csv,exe,aspx"
	gState, urlSlice := preSetupTest(cfg, "2010", finished, t)
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/a.exe"]; !ok || x == nil {
		t.Error("Failed Ext test1")
	}

	if x, ok := found["/a.aspx"]; !ok || x == nil {
		t.Error("Failed Ext test2")
	}

	if x, ok := found["/a.csv"]; !ok || x == nil {
		t.Error("Failed Ext test3")
	}
}

func TestAllOut(t *testing.T) {
	// sets all responses as 'found' for the purposes of output. Should not pass any logic tests for adding additional URLs etc
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.ShowAll = true
	gState, urlsSlice := preSetupTest(cfg, "2011", finished, t)
	found := postSetupTest(urlsSlice, gState)
	gState.Wait()

	//Check the 404's were received and set as found
	if x, ok := found["/a/x/c"]; ok && x != nil {
		t.Error("Failed OutAll Test 1, performed check on non-existent prefix")
	}

	if x, ok := found["/x"]; ok && x != nil {
		//have 404 response in found set
		if x.StatusCode != 404 {
			t.Error("Failed OutAll Test 3, got unexpected response recorded for 404 check")
		}
	} else {
		//didn't have '/a/x' (soft 404) in found set
		t.Error("Failed OutAll Test 2, did not have 404 response in found set")
	}
}

func TestCanary(t *testing.T) {
	// check that the canary response is respected, and set as the soft-404 check
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Canary = "canarystringvalue"
	gState, urlsSlice := preSetupTest(cfg, "2012", finished, t)
	gState.WordList = append(gState.WordList, "canarystringvalue")
	gState.WordList = append(gState.WordList, "canarysimilar")
	found := postSetupTest(urlsSlice, gState)
	gState.Wait()

	//check canary value didn't somehow work
	if x, ok := found["/canarystringvalue"]; ok && x != nil {
		t.Error("Failed Canary Test 1, server responded good to canarystringvalue (and was found??)")
	}

	//check canarysimilar is not set as valid response
	if x, ok := found["/canarysimilar"]; ok && x != nil {
		t.Error("Failed Canary Test 2, server responded good to canarystringvalue (and was found??)")
	}

	//since we set the canary to something different, we should record a found for the other soft-404's
	if x, ok := found["/a/x"]; !ok || x == nil {
		t.Error("Failed Canary Test 3, didn't find old soft 404 (/a/x)")
	}

	if x, ok := found["/a/y"]; !ok || x == nil {
		t.Error("Failed Canary Test 4, didn't find old modified soft 404 (/a/y)")
	}

}

func TestHeaders(t *testing.T) {
	//check for custom header workyness
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Headers = recursebuster.ArrayStringFlag{}
	cfg.Headers.Set("X-ATT-DeviceId:XXXXX")
	gState, urlSlice := preSetupTest(cfg, "2013", finished, t)
	gState.WordList = append(gState.WordList, "customheaderonly")
	gState.WordList = append(gState.WordList, "onlynocustomheader")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/customheaderonly"]; !ok || x == nil {
		t.Error("Failed Custom header check 1, didn't find a path it should have")
	}

	if x, ok := found["/onlynocustomheader"]; ok && x != nil {
		t.Error("Failed Custom header check 2, found the path it shouldn't have")
	}

}

func TestNoGET(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.NoGet = true
	gState, urlSlice := preSetupTest(cfg, "2014", finished, t)
	gState.WordList = append(gState.WordList, "getonly")
	gState.WordList = append(gState.WordList, "headonly")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/headonly"]; !ok || x == nil {
		t.Error("Failed check 1, didn't find a path it should have")
	}

	if x, ok := found["/getonly"]; ok && x != nil {
		t.Error("Failed check 2, found the path it shouldn't have")
	}
}

func TestNoHEAD(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.NoHead = true
	gState, urlSlice := preSetupTest(cfg, "2015", finished, t)
	gState.WordList = append(gState.WordList, "getonly")
	gState.WordList = append(gState.WordList, "headonly")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/headonly"]; ok && x != nil {
		t.Error("Failed check 2, found the path it shouldn't have")
	}

	if x, ok := found["/getonly"]; !ok || x == nil {
		t.Error("Failed check 1, didn't find a path it should have")
	}
}

func TestNoRecursion(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.NoRecursion = true
	gState, urlSlice := preSetupTest(cfg, "2016", finished, t)
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/a/b"]; ok && x != nil {
		t.Error("Failed check 1, found a path it shouldn't have")
	}
	//todo: add more recursive things here to make sure it's not doing the thing
}

func TestNoSpider(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.NoSpider = true
	gState, urlSlice := preSetupTest(cfg, "2017", finished, t)
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/spideronly"]; ok && x != nil {
		t.Error("Failed check 1, found a path it shouldn't have")
	}
}

func TestNoWildcard(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.NoWildcardChecks = true
	gState, urlSlice := preSetupTest(cfg, "2018", finished, t)
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	//should find /a/x and /a/y
	if x, ok := found["/a/x"]; !ok || x == nil {
		t.Error("Failed check 1, didn't find a path it should have")
	}

	if x, ok := found["/a/y"]; !ok || x == nil {
		t.Error("Failed check 2, didn't find a path it should have")
	}
}

func TestMethods(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Methods = "GET,POST,CATPICTURESPLEASE"
	gState, urlSlice := preSetupTest(cfg, "2019", finished, t)
	gState.WordList = append(gState.WordList, "postonly")
	gState.WordList = append(gState.WordList, "catpicturesmethod")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	//should find /a/x and /a/y
	if x, ok := found["/postonly"]; !ok || x == nil {
		t.Error("Failed check 1, didn't find a path it should have")
	}

	if x, ok := found["/catpicturesmethod"]; !ok || x == nil {
		t.Error("Failed check 2, didn't find a path it should have")
	}
}

func TestRobots(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	gState, urlSlice := preSetupTest(cfg, "2020", finished, t)
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	//should find
	if x, ok := found["/robotsfolder/a"]; !ok || x == nil {
		t.Error("Failed check 1, didn't find a path it should have")
	}

	if x, ok := found["/junk/a"]; !ok || x == nil {
		t.Error("Failed check 2, didn't find a path it should have")
	}

	if x, ok := found["/robotsfolder/b"]; !ok || x == nil {
		t.Error("Failed check 3, didn't find a path it should have")
	}

}

func TestWeirdWords(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	gState, urlSlice := preSetupTest(cfg, "2021", finished, t)
	//add some woderful and weird things to the wordlist
	for i := 0; i < 256; i++ {
		gState.WordList = append(gState.WordList, "te"+string(i)+"st")
		//string(i)
	}
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	//same as the regular test
	//check for each specific line that should be in there..
	tested := []string{}
	ok := []string{
		"/a", "/a/b", "/a/b/c", "/a/", "/spideronly",
		"/b", "/b/c",
		"/a/b/c/", "/a/b/c/d",
		"/c/d", "/c", "/c/",
	}
	for _, i := range ok {
		tested = append(tested, i)
		if x, ok := found[i]; !ok || x == nil {
			t.Error("Did not find " + i)
		}
	}
}

func TestGoodCodes(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.GoodResponses = "500"
	gState, urlSlice := preSetupTest(cfg, "2022", finished, t)
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	tested := []string{}
	ok := []string{
		"/c", "/c/",
	}
	for _, i := range ok {
		tested = append(tested, i)
		if x, ok := found[i]; !ok || x == nil {
			t.Error("Did not find " + i)
		}
	}

	//check for values that should not have been found
	for k := range found {
		if strings.Contains(k, "z") {
			t.Error("Found (but should not have) " + k)
		}
	}

	if x, ok := found["/a/x"]; ok && x != nil {
		t.Error("Found (but should not have) /a/x")
	}

	if x, ok := found["/a"]; ok && x != nil {
		t.Error("Found (but should not have) /a")
	}

	if x, ok := found["/a/b"]; ok && x != nil {
		t.Error("Found (but should not have) /a/b")
	}

	if x, ok := found["/b/y"]; ok && x != nil {
		t.Error("Found (but should not have) /b/y")
	}

}

func TestVHost(t *testing.T) {
	t.Parallel()
	finished := make(chan struct{})
	cfg := getDefaultConfig()
	cfg.Vhost = "vhost"
	gState, urlSlice := preSetupTest(cfg, "2023", finished, t)
	gState.WordList = append(gState.WordList, "vhost1")
	found := postSetupTest(urlSlice, gState)
	gState.Wait()

	if x, ok := found["/vhost1"]; !ok || x == nil {
		t.Error("Failed check 1, didn't find a path it should have")
	}

}

func postSetupTest(urlSlice []string, gState *recursebuster.State) (found map[string]*http.Response) {
	//start up the management goroutines
	go gState.ManageRequests()
	go gState.ManageNewURLs()
	go gState.StartTestWorker() //single thread only - todo: when doing multithread tests make this gooder

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
	//gState.Chans.GetWorkers() <- struct{}{}
	go gState.StartBusting(randURL, *u)

	//start the print channel (so that we can see output if a test fails)
	pc := gState.Chans.PrintChan()
	go func() {
		for {
			p := <-pc
			gState.DoneWG()
			if p.Content != "" {

			}
			//fmt.Println(p.Content)
		}
	}()

	//use the found map to determine later on if we have found the expected URL's
	found = make(map[string]*http.Response)
	conf := gState.Chans.ConfirmedChan()
	go func() {
		t := time.NewTicker(1 * time.Second).C
		for {
			select {
			case x := <-conf:

				u, e := url.Parse(x.URL)
				if e != nil {
					gState.DoneWG()
					panic(e)
				}
				found[u.Path] = x.Result
				gState.DoneWG()
				//fmt.Println("CONFIRMED!", x)
			case <-t:
				//fmt.Println(gState.wg)
			}
		}
	}()
	return
}

func preSetupTest(cfg *recursebuster.Config, servPort string, finished chan struct{}, t *testing.T) (stateObject *recursebuster.State, urlSlice []string) {
	//Test default functions. Basic dirb should work, and all files should be found as expected

	//basic state setup
	globalState := recursebuster.State{}.Init()
	if cfg != nil {
		globalState.Cfg = cfg
	}
	globalState.Hosts.Init()

	//start the test server
	setup := make(chan struct{})
	go testserver.TestServer{}.Start(servPort, finished, setup, t)
	<-setup //whoa, concurrency sucks???
	//test URL
	globalState.Cfg.URL = localURL + servPort

	//default slice starter-upper
	urlSlice = cmd.GetURLSlice(globalState)

	//setup the config to default values
	setupConfig(globalState, urlSlice[0], cfg)

	//normal main setup state call
	globalState.SetupState()

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
	stateObject = globalState
	return
}

func getDefaultConfig() *recursebuster.Config {
	return &recursebuster.Config{
		Version: "TEST",
		ShowAll: false,

		Agent:             "RecurseBuster/" + "TESTING",
		AppendDir:         true,
		Auth:              "",
		BadHeader:         nil, //ArrayStringFlag{} // "" // "Check for presence of this header. If an exact match is found"
		BadResponses:      "404",
		BlacklistLocation: "",
		BurpMode:          false,
		Canary:            "",
		CleanOutput:       false,
		Cookies:           "",
		Debug:             false,
		Extensions:        "",
		FollowRedirects:   false,
		Headers:           nil, // "Additional headers to include with request. Supply as key:value. Can specify multiple - eg '-headers X-Forwarded-For:127.0.01 -headers X-ATT-DeviceId:XXXXX'")
		HTTPS:             false,
		InputList:         "",
		Localpath:         "." + string(os.PathSeparator) + "busted.txt",
		Methods:           "GET",
		NoBase:            false,
		NoGet:             false,
		NoHead:            false,
		NoRecursion:       false,
		NoSpider:          false,
		NoStartStop:       false,
		NoStatus:          false,
		NoUI:              true,
		NoWildcardChecks:  false,
		ProxyAddr:         "",
		Ratio404:          0.95,
		ShowLen:           false,
		ShowVersion:       false,
		SSLIgnore:         false,
		Threads:           1,
		Timeout:           20,
		VerboseLevel:      0,
		WhitelistLocation: "",
		Wordlist:          "",

		URL: localURL,
	}

}

func setupConfig(globalState *recursebuster.State, urlSliceZero string, cfg *recursebuster.Config) {

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
		globalState.Cfg.Canary = recursebuster.RandString()
	}
}
