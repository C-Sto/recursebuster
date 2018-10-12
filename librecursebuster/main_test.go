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

const localURL = "http://localhost:12345/"

func TestBasicFunctionality(t *testing.T) {

	//the state should probably change per different host.. eventually
	globalState := State{}.Init()
	globalState.Hosts.Init()

	testserver.Start()

	globalState.Cfg.URL = localURL

	urlSlice := getURLSlice(globalState)

	setupConfig(globalState, urlSlice[0])

	SetupState(globalState)

	wordlist := `a
b
c
d
e
x
y
z
`

	globalState.WordList = strings.Split(wordlist, "\n")

	globalState.Cfg.Wordlist = "test"

	globalState.DirbProgress = new(uint32)

	go ManageRequests()
	go ManageNewURLs()

	u, err := url.Parse(urlSlice[0])
	if err != nil {
		panic(err)
	}
	//canary
	prefix := u.String()
	if len(prefix) > 0 && string(prefix[len(prefix)-1]) != "/" {
		prefix = prefix + "/"
	}
	randURL := fmt.Sprintf("%s%s", prefix, globalState.Cfg.Canary)
	globalState.AddWG()
	gState.Chans.GetWorkers() <- struct{}{}
	go StartBusting(randURL, *u)

	go func() {
		for {
			p := <-gState.Chans.printChan
			globalState.wg.Done()
			if p.Level > 1 {
				continue
			}
		}
	}()
	found := make(map[string]bool)
	go func() {
		t := time.NewTicker(1 * time.Second).C
		for {
			select {
			case x := <-gState.Chans.confirmedChan:
				globalState.wg.Done()
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
	//waitgroup check (if test times out, the waitgroup is broken... somewhere)
	globalState.Wait()

	//fmt.Println("Ready to test!")
	//check for each specific line that should be in there..
	ok200 := []string{
		"/a", "/a/b", "/a/b/c", "/a/",
	}
	for _, i := range ok200 {
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}
	ok300 := []string{
		"/b", "/b/c",
	}
	for _, i := range ok300 {
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}
	ok400 := []string{
		"/a/b/c/", "/a/b/c/d",
	}
	for _, i := range ok400 {
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}
	ok500 := []string{
		"/c/d", "/c", "/c/",
	}
	for _, i := range ok500 {
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}

}

func setupConfig(globalState *State, urlSliceZero string) {
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
