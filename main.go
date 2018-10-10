package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	_ "net/http/pprof"

	"github.com/c-sto/recursebuster/librecursebuster"
	"github.com/fatih/color"
)

const version = "1.5.7"

func main() {
	if runtime.GOOS == "windows" { //lol goos
		//can't use color.Error, because *nix etc don't have that for some reason :(
		librecursebuster.InitLogger(color.Output, color.Output, color.Output, color.Output, color.Output, color.Output, color.Output, color.Output, color.Output, color.Output)
	} else {
		librecursebuster.InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	}

	cfg := &librecursebuster.Config{}

	//the state should probably change per different host.. eventually
	globalState := librecursebuster.State{}.Init()
	globalState.Hosts.Init()

	cfg.Version = version
	totesTested := uint64(0)
	globalState.TotalTested = &totesTested
	flag.BoolVar(&cfg.ShowAll, "all", false, "Show, and write the result of all checks")
	flag.BoolVar(&cfg.AppendDir, "appendslash", false, "Append a / to all directory bruteforce requests (like extension, but slash instead of .yourthing)")
	flag.StringVar(&cfg.Auth, "auth", "", "Basic auth. Supply this with the base64 encoded portion to be placed after the word 'Basic' in the Authorization header.")
	flag.StringVar(&cfg.BadResponses, "bad", "404", "Responses to consider 'bad' or 'not found'. Comma-separated This works the opposite way of gobuster!")
	flag.Var(&cfg.BadHeader, "badheader", "Check for presence of this header. If an exact match is found, the response is considered bad.Supply as key:value. Can specify multiple - eg '-badheader Location:cats -badheader X-ATT-DeviceId:XXXXX'")
	//flag.StringVar(&cfg.BodyContent, "body", "", "File containing content to send in the body of the request.") all empty body for now
	flag.StringVar(&cfg.BlacklistLocation, "blacklist", "", "Blacklist of prefixes to not check. Will not check on exact matches.")
	flag.StringVar(&cfg.Canary, "canary", "", "Custom value to use to check for wildcards")
	flag.BoolVar(&cfg.CleanOutput, "clean", false, "Output clean URLs to the output file for easy loading into other tools and whatnot.")
	flag.StringVar(&cfg.Cookies, "cookies", "", "Any cookies to include with requests. This is smashed into the cookies header, so copy straight from burp I guess.")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debugging")
	//flag.IntVar(&cfg.MaxDirs, "dirs", 1, "Maximum directories to perform busting on concurrently forcing limit to 1 because it's complicated otherwise
	flag.StringVar(&cfg.Extensions, "ext", "", "Extensions to append to checks. Multiple extensions can be specified, comma separate them.")
	flag.Var(&cfg.Headers, "headers", "Additional headers to include with request. Supply as key:value. Can specify multiple - eg '-headers X-Forwarded-For:127.0.01 -headers X-ATT-DeviceId:XXXXX'")
	flag.BoolVar(&cfg.HTTPS, "https", false, "Use HTTPS instead of HTTP.")
	flag.StringVar(&cfg.InputList, "iL", "", "File to use as an input list of URL's to start from")
	flag.BoolVar(&cfg.SSLIgnore, "k", false, "Ignore SSL check")
	flag.BoolVar(&cfg.ShowLen, "len", false, "Show, and write the length of the response")
	flag.BoolVar(&cfg.NoBase, "nobase", false, "Don't perform a request to the base URL")
	flag.BoolVar(&cfg.NoGet, "noget", false, "Do not perform a GET request (only use HEAD request/response)")
	flag.BoolVar(&cfg.NoHead, "nohead", false, "Don't optimize GET requests with a HEAD (only send the GET)")
	flag.BoolVar(&cfg.NoRecursion, "norecursion", false, "Disable recursion, just work on the specified directory. Also disables spider function.")
	flag.BoolVar(&cfg.NoSpider, "nospider", false, "Don't search the page body for links, and directories to add to the spider queue.")
	flag.BoolVar(&cfg.NoStatus, "nostatus", false, "Don't print status info (for if it messes with the terminal)")
	flag.BoolVar(&cfg.NoStartStop, "nostartstop", false, "Don't show start/stop info messages")
	flag.BoolVar(&cfg.NoWildcardChecks, "nowildcard", false, "Don't perform wildcard checks for soft 404 detection")
	flag.BoolVar(&cfg.NoUI, "noui", false, "Don't use sexy ui")
	flag.StringVar(&cfg.Localpath, "o", "."+string(os.PathSeparator)+"busted.txt", "Local file to dump into")
	flag.StringVar(&cfg.Methods, "methods", "GET", "Methods to use for checks. Multiple methods can be specified, comma separate them. Requests will be sent with an empty body (unless body is specified)")
	flag.StringVar(&cfg.ProxyAddr, "proxy", "", "Proxy configuration options in the form ip:port eg: 127.0.0.1:9050. Note! If you want this to work with burp/use it with a HTTP proxy, specify as http://ip:port")
	flag.Float64Var(&cfg.Ratio404, "ratio", 0.95, "Similarity ratio to the 404 canary page.")
	flag.BoolVar(&cfg.FollowRedirects, "redirect", false, "Follow redirects")
	flag.BoolVar(&cfg.BurpMode, "sitemap", false, "Send 'good' requests to the configured proxy. Requires the proxy flag to be set. ***NOTE: with this option, the proxy is ONLY used for good requests - all other requests go out as normal!***")
	flag.IntVar(&cfg.Threads, "t", 1, "Number of concurrent threads")
	flag.IntVar(&cfg.Timeout, "timeout", 20, "Timeout (seconds) for HTTP/TCP connections")
	flag.StringVar(&cfg.URL, "u", "", "Url to spider")
	flag.StringVar(&cfg.Agent, "ua", "RecurseBuster/"+version, "User agent to use when sending requests.")
	flag.IntVar(&cfg.VerboseLevel, "v", 0, "Verbosity level for output messages.")
	flag.BoolVar(&cfg.ShowVersion, "version", false, "Show version number and exit")
	flag.StringVar(&cfg.Wordlist, "w", "", "Wordlist to use for bruteforce. Blank for spider only")
	flag.StringVar(&cfg.WhitelistLocation, "whitelist", "", "Whitelist of domains to include in brute-force")

	flag.Parse()

	if cfg.ShowVersion {
		librecursebuster.PrintBanner(cfg)
		os.Exit(0)
	}

	if cfg.URL == "" && cfg.InputList == "" {
		flag.Usage()
		os.Exit(1)
	}

	urlSlice := getURLSlice(cfg)

	setupConfig(cfg, globalState, urlSlice[0])

	setupState(globalState, cfg)

	//do first load of urls (send canary requests to make sure we can dirbust them)
	quitChan := make(chan struct{})
	if !cfg.NoUI {
		uiWG := &sync.WaitGroup{}
		uiWG.Add(1)
		go uiQuit(quitChan)
		go globalState.StartUI(uiWG, quitChan)
		uiWG.Wait()
	}

	if cfg.NoUI {
		librecursebuster.PrintBanner(cfg)
		go librecursebuster.StatusPrinter(cfg)
	} else {
		go librecursebuster.UIPrinter(cfg)
	}
	go librecursebuster.ManageRequests(cfg)
	go librecursebuster.ManageNewURLs(cfg)
	go librecursebuster.OutputWriter(cfg, cfg.Localpath)
	go librecursebuster.StatsTracker()

	librecursebuster.PrintOutput("Starting recursebuster...     ", librecursebuster.Info, 0)

	//seed the workers
	for _, s := range urlSlice {
		u, err := url.Parse(s)
		if err != nil {
			panic(err)
		}

		if u.Scheme == "" {
			if cfg.HTTPS {
				u, err = url.Parse("https://" + s)
			} else {
				u, err = url.Parse("http://" + s)
			}
		}
		if err != nil {
			//this should never actually happen
			panic(err)
		}

		//do canary etc

		prefix := u.String()
		if len(prefix) > 0 && string(prefix[len(prefix)-1]) != "/" {
			prefix = prefix + "/"
		}
		randURL := fmt.Sprintf("%s%s", prefix, cfg.Canary)
		globalState.Chans.GetWorkers() <- struct{}{}
		globalState.AddWG()
		go librecursebuster.StartBusting(cfg, randURL, *u)

	}

	//wait for completion
	globalState.Wait()

}

func getURLSlice(cfg *librecursebuster.Config) []string {
	urlSlice := []string{}
	if cfg.URL != "" {
		urlSlice = append(urlSlice, cfg.URL)
	}

	if cfg.InputList != "" { //can have both -u flag and -iL flag
		//must be using an input list
		URLList := make(chan string, 10)
		go librecursebuster.LoadWords(cfg.InputList, URLList)
		for x := range URLList {
			//ensure all urls will parse good
			_, err := url.Parse(x)
			if err != nil {
				panic("URL parse fail: " + err.Error())
			}
			urlSlice = append(urlSlice, x)
			//globalState.Whitelist[u.Host] = true
		}
	}

	return urlSlice
}

func uiQuit(quitChan chan struct{}) {
	<-quitChan
	os.Exit(0)
}

func setupState(globalState *librecursebuster.State, cfg *librecursebuster.Config) {
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

	globalState.Client = librecursebuster.ConfigureHTTPClient(cfg, false)
	globalState.BurpClient = librecursebuster.ConfigureHTTPClient(cfg, true)

	globalState.Version = cfg.Version

	if cfg.BlacklistLocation != "" {
		readerChan := make(chan string, 100)
		go librecursebuster.LoadWords(cfg.BlacklistLocation, readerChan)
		for x := range readerChan {
			globalState.Blacklist[x] = true
		}
	}

	if cfg.WhitelistLocation != "" {
		readerChan := make(chan string, 100)
		go librecursebuster.LoadWords(cfg.WhitelistLocation, readerChan)
		for x := range readerChan {
			globalState.Whitelist[x] = true
		}
	}

	if cfg.Wordlist != "" { // && cfg.MaxDirs == 1 {

		zerod := uint32(0)
		globalState.DirbProgress = &zerod

		//	zero := uint32(0)
		//	globalState.WordlistLen = &zero

		readerChan := make(chan string, 100)
		go librecursebuster.LoadWords(cfg.Wordlist, readerChan)
		for v := range readerChan {
			globalState.WordList = append(globalState.WordList, v)
			//atomic.AddUint32(globalState.WordlistLen, 1)
		}
	}
	librecursebuster.SetState(globalState)
}

func setupConfig(cfg *librecursebuster.Config, globalState *librecursebuster.State, urlSliceZero string) {
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
		cfg.Canary = librecursebuster.RandString()
	}

}
