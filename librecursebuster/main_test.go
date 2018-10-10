package librecursebuster

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c-sto/recursebuster/librecursebuster/testserver"
)

const localURL = "http://localhost:12345/"

func TestBasicFunctionality(t *testing.T) {

	cfg := &Config{}

	//the state should probably change per different host.. eventually
	globalState := State{}.Init()
	globalState.Hosts.Init()

	testserver.Start()

	cfg.URL = localURL

	urlSlice := getURLSlice(cfg)

	setupConfig(cfg, globalState, urlSlice[0])

	setupState(globalState, cfg)

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

	cfg.Wordlist = "test"

	go ManageRequests(cfg)
	go ManageNewURLs(cfg)

	u, err := url.Parse(urlSlice[0])
	if err != nil {
		panic(err)
	}
	//canary
	prefix := u.String()
	if len(prefix) > 0 && string(prefix[len(prefix)-1]) != "/" {
		prefix = prefix + "/"
	}
	randURL := fmt.Sprintf("%s%s", prefix, cfg.Canary)
	globalState.AddWG()
	gState.Chans.GetWorkers() <- struct{}{}
	go StartBusting(cfg, randURL, *u)

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

func setupConfig(cfg *Config, globalState *State, urlSliceZero string) {
	cfg.Version = "TEST"
	totesTested := uint64(0)
	globalState.TotalTested = &totesTested
	cfg.ShowAll = false
	cfg.AppendDir = true
	cfg.Auth = ""
	cfg.BadResponses = "404"
	//cfg.BadHeader, "Check for presence of this header. If an exact match is found"
	//cfg.BodyContent, ""
	cfg.BlacklistLocation = ""
	cfg.Canary = ""
	cfg.CleanOutput = false
	cfg.Cookies = ""
	cfg.Debug = false
	//cfg.MaxDirs = 1
	cfg.Extensions = ""
	//cfg.Headers = "Additional headers to include with request. Supply as key:value. Can specify multiple - eg '-headers X-Forwarded-For:127.0.01 -headers X-ATT-DeviceId:XXXXX'")
	cfg.HTTPS = false
	cfg.InputList = ""
	cfg.SSLIgnore = false
	cfg.ShowLen = false
	cfg.NoBase = false
	cfg.NoGet = false
	cfg.NoHead = false
	cfg.NoRecursion = false
	cfg.NoSpider = false
	cfg.NoStatus = false
	cfg.NoStartStop = false
	cfg.NoWildcardChecks = false
	cfg.NoUI = true
	cfg.Localpath = "." + string(os.PathSeparator) + "busted.txt"
	cfg.Methods = "GET"
	cfg.ProxyAddr = ""
	cfg.Ratio404 = 0.95
	cfg.FollowRedirects = false
	cfg.BurpMode = false
	cfg.Threads = 1
	cfg.Timeout = 20
	cfg.URL = ""
	cfg.Agent = "RecurseBuster/" + cfg.Version
	cfg.VerboseLevel = 0
	cfg.ShowVersion = false
	cfg.Wordlist = ""
	cfg.WhitelistLocation = ""

	cfg.URL = localURL

	var h *url.URL
	var err error
	h, err = url.Parse(urlSliceZero)
	if err != nil {
		panic(err)
	}

	if h.Scheme == "" {
		if cfg.HTTPS {
			h, err = url.Parse("https://" + urlSliceZero)
		} else {
			h, err = url.Parse("http://" + urlSliceZero)
		}
	}
	if err != nil {
		panic(err)
	}
	globalState.Hosts.AddHost(h)

	if cfg.Canary == "" {
		cfg.Canary = RandString()
	}

}

func setupState(globalState *State, cfg *Config) {
	for _, x := range strings.Split(cfg.Extensions, ",") {
		globalState.Extensions = append(globalState.Extensions, x)
	}

	for _, x := range strings.Split(cfg.Methods, ",") {
		globalState.Methods = append(globalState.Methods, x)
	}

	for _, x := range strings.Split(cfg.BadResponses, ",") {
		i, err := strconv.Atoi(x)
		if err != nil {
			panic(err)
		}
		globalState.BadResponses[i] = true //this is probably a candidate for individual urls. Unsure how to config that cleanly though
	}

	globalState.Client = ConfigureHTTPClient(cfg, false)
	globalState.BurpClient = ConfigureHTTPClient(cfg, true)

	globalState.Version = cfg.Version

	// && cfg.MaxDirs == 1 {

	zerod := uint32(0)
	globalState.DirbProgress = &zerod

	//	zero := uint32(0)
	//	globalState.WordlistLen = &zero

	globalState.StartTime = time.Now()
	globalState.PerSecondShort = new(uint64)
	globalState.PerSecondLong = new(uint64)

	SetState(globalState)
}
func getURLSlice(cfg *Config) []string {
	urlSlice := []string{}
	if cfg.URL != "" {
		urlSlice = append(urlSlice, cfg.URL)
	}
	return urlSlice
}
