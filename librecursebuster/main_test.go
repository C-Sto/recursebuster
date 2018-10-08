package librecursebuster

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c-sto/recursebuster/librecursebuster/testserver"
)

func TestBasicFunctionality(t *testing.T) {

	wg := &sync.WaitGroup{}
	cfg := &Config{}

	//the state should probably change per different host.. eventually
	globalState := &State{
		BadResponses: make(map[int]bool),
		Whitelist:    make(map[string]bool),
		Blacklist:    make(map[string]bool),
	}
	globalState.Hosts.Init()

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
	cfg.NoUI = false
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

	testserver.Start()
	localURL := "http://localhost:12345/"
	cfg.URL = localURL
	wordlist := `a
b
c
d
e
x
y
z
`
	printChan := make(chan OutLine, 50)

	urlSlice := getURLSlice(cfg, printChan)

	setupConfig(cfg, globalState, urlSlice[0], printChan)

	setupState(globalState, cfg, wg, printChan)

	//setup channels
	pages := make(chan SpiderPage, 1000)
	newPages := make(chan SpiderPage, 10000)
	confirmed := make(chan SpiderPage, 1000)
	workers := make(chan struct{}, cfg.Threads)
	//maxDirs := make(chan struct{}, cfg.MaxDirs)
	testChan := make(chan string, 100)
	//quitChan := make(chan struct{})

	globalState.StartTime = time.Now()
	globalState.PerSecondShort = new(uint64)
	globalState.PerSecondLong = new(uint64)

	globalState.WordList = strings.Split(wordlist, "\n")

	cfg.Wordlist = "test"

	go ManageNewURLs(cfg, wg, pages, newPages, printChan)
	go ManageRequests(cfg, wg, pages, newPages, confirmed, workers, printChan, testChan)

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
	wg.Add(1)
	workers <- struct{}{}
	go startBusting(wg, globalState, cfg, workers, printChan, pages, randURL, *u)

	go func() {
		for {
			p := <-printChan
			if p.Level > 1 {
				continue
			}
			fmt.Println(p.Content)
		}
	}()
	found := make(map[string]bool)
	go func() {
		for x := range confirmed {
			found[x.URL] = true
			fmt.Println("CONFIRMED!", x)
		}
	}()
	//waitgroup check (if test times out, the waitgroup is broken... somewhere)
	wg.Wait()

	//check for each specific line that should be in there..
	/*
			200 (OK)
		/a
		/a/b
		/a/b/c
		/a/
		/a/x (200, but same body as /x (404))
		/a/y (200, but very similar body to /x (404))
	*/
	// /
	for _, i := range strings.Split(wordlist, "\n") {
		if x, ok := found[i]; !ok || !x {
			panic("Did not find " + i)
		}
	}
	// /a
	/*
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
	   /c/
	*/
	panic("no")
}

func setupConfig(cfg *Config, globalState *State, urlSliceZero string, printChan chan OutLine) {
	if cfg.Debug {
		go func() {
			http.ListenAndServe("localhost:6061", http.DefaultServeMux)
		}()
	}

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
		cfg.Canary = RandString(printChan)
	}

}

func setupState(globalState *State, cfg *Config, wg *sync.WaitGroup, printChan chan OutLine) {
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
	globalState.Client = ConfigureHTTPClient(cfg, wg, printChan, false)
	globalState.BurpClient = ConfigureHTTPClient(cfg, wg, printChan, true)

	globalState.StopDir = make(chan struct{}, 1)
	globalState.CMut = &sync.RWMutex{}
	globalState.Checked = make(map[string]bool)
	globalState.Version = cfg.Version

	if cfg.BlacklistLocation != "" {
		readerChan := make(chan string, 100)
		go LoadWords(cfg.BlacklistLocation, readerChan, printChan)
		for x := range readerChan {
			globalState.Blacklist[x] = true
		}
	}

	if cfg.WhitelistLocation != "" {
		readerChan := make(chan string, 100)
		go LoadWords(cfg.WhitelistLocation, readerChan, printChan)
		for x := range readerChan {
			globalState.Whitelist[x] = true
		}
	}

	zerod := uint32(0)
	globalState.DirbProgress = &zerod

	SetState(globalState)
}

func getURLSlice(cfg *Config, printChan chan OutLine) []string {
	urlSlice := []string{}
	if cfg.URL != "" {
		urlSlice = append(urlSlice, cfg.URL)
	}
	return urlSlice
}

func startBusting(wg *sync.WaitGroup, globalState *State, cfg *Config, workers chan struct{}, printChan chan OutLine, pages chan SpiderPage, randURL string, u url.URL) {
	defer wg.Done()
	if !cfg.NoWildcardChecks {
		resp, err := HTTPReq("GET", randURL, globalState.Client, cfg)
		<-workers
		if err != nil {
			if cfg.InputList != "" {
				PrintOutput(
					err.Error(),
					Error,
					0,
					wg,
					printChan,
				)
				return
			}
			panic("Canary Error, check url is correct: " + randURL + "\n" + err.Error())

		}
		PrintOutput(
			fmt.Sprintf("Canary sent: %s, Response: %v", randURL, resp.Status),
			Debug, 2, wg, printChan,
		)
		content, _ := ioutil.ReadAll(resp.Body)
		globalState.Hosts.AddSoft404Content(u.Host, content, resp) // Soft404ResponseBody = xx
	} else {
		<-workers
	}
	x := SpiderPage{}
	x.URL = u.String()
	x.Reference = &u

	globalState.CMut.Lock()
	defer globalState.CMut.Unlock()
	if ok := globalState.Checked[u.String()+"/"]; !strings.HasSuffix(u.String(), "/") && !ok {
		wg.Add(1)
		pages <- SpiderPage{
			URL:       u.String() + "/",
			Reference: &u,
		}
		globalState.Checked[u.String()+"/"] = true
		PrintOutput("URL Added: "+u.String()+"/", Debug, 3, wg, printChan)
	}
	if ok := globalState.Checked[x.URL]; !ok {
		wg.Add(1)
		pages <- x
		globalState.Checked[x.URL] = true
		PrintOutput("URL Added: "+x.URL, Debug, 3, wg, printChan)
	}
}
