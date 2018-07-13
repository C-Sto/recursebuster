package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/c-sto/recursebuster/librecursebuster"
	"github.com/fatih/color"
	"golang.org/x/net/proxy"
)

const version = "1.0.8"

func main() {
	if runtime.GOOS == "windows" { //lol goos
		//can't use color.Error, because *nix etc don't have that for some reason :(
		librecursebuster.InitLogger(color.Output, color.Output, color.Output, color.Output, color.Output, color.Output, color.Output, color.Output, color.Output, color.Output)
	} else {
		librecursebuster.InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	}
	cfg := librecursebuster.Config{}
	seedPages := []librecursebuster.SpiderPage{}
	//the state should probably change per different host.. eventually
	state := librecursebuster.State{
		BadResponses: make(map[int]bool),
		Whitelist:    make(map[string]bool),
		Blacklist:    make(map[string]bool),
	}
	//states := []librecursebuster.State{state}

	cfg.Version = version
	totesTested := uint64(0)
	state.TotalTested = &totesTested
	showVersion := true
	flag.IntVar(&cfg.Threads, "t", 1, "Number of concurrent threads")
	flag.StringVar(&cfg.URL, "u", "", "Url to spider")
	flag.StringVar(&cfg.Localpath, "o", "."+string(os.PathSeparator)+"busted.txt", "Local file to dump into")
	flag.BoolVar(&cfg.SSLIgnore, "k", false, "Ignore SSL check")
	flag.StringVar(&cfg.ProxyAddr, "p", "", "Proxy configuration options in the form ip:port eg: 127.0.0.1:9050. Note! If you want this to work with burp/use it with a HTTP proxy, specify as http://ip:port")
	flag.StringVar(&cfg.Wordlist, "w", "", "Wordlist to use for bruteforce. Blank for spider only")
	flag.StringVar(&cfg.Canary, "canary", "", "Custom value to use to check for wildcards")
	flag.StringVar(&cfg.Agent, "ua", "RecurseBuster/"+version, "User agent to use when sending requests.")
	flag.Float64Var(&cfg.Ratio404, "ratio", 0.95, "Similarity ratio to the 404 canary page.")
	flag.StringVar(&cfg.BlacklistLocation, "blacklist", "", "Blacklist of prefixes to not check. Will not check on exact matches.")
	flag.BoolVar(&cfg.NoSpider, "nospider", false, "Don't search the page body for links, and directories to add to the spider queue.")
	flag.IntVar(&cfg.Timeout, "timeout", 20, "Timeout (seconds) for HTTP/TCP connections")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debugging")
	flag.BoolVar(&cfg.NoGet, "noget", false, "Do not perform a GET request (only use HEAD request/response)")
	flag.IntVar(&cfg.MaxDirs, "dirs", 1, "Maximum directories to perform busting on concurrently NOTE: directories will still be brute forced, this setting simply directs how many should be concurrently bruteforced")
	flag.BoolVar(&cfg.ShowAll, "all", false, "Show, and write the result of all checks")
	flag.BoolVar(&cfg.ShowLen, "len", false, "Show, and write the length of the response")
	flag.StringVar(&cfg.WhitelistLocation, "whitelist", "", "Whitelist of domains to include in brute-force")
	flag.BoolVar(&cfg.FollowRedirects, "redirect", false, "Follow redirects")
	flag.StringVar(&cfg.BadResponses, "bad", "404", "Responses to consider 'bad' or 'not found'. Comma-separated This works the opposite way of gobuster!")
	flag.BoolVar(&cfg.CleanOutput, "clean", false, "Output clean URLs to the output file for easy loading into other tools and whatnot.")
	flag.StringVar(&cfg.Cookies, "cookies", "", "Any cookies to include with requests. This is smashed into the cookies header, so copy straight from burp I guess.")
	flag.StringVar(&cfg.Extensions, "ext", "", "Extensions to append to checks. Multiple extensions can be specified, comma separate them.")
	// soon.jpg	flag.StringVar(&cfg.InputList, "iL", "", "File to use as an input list of URL's to start from")
	flag.BoolVar(&cfg.HTTPS, "https", false, "Use HTTPS instead of HTTP.")
	flag.IntVar(&cfg.VerboseLevel, "v", 0, "Verbosity level for output messages.")
	flag.BoolVar(&showVersion, "version", false, "Show version number and exit")
	flag.BoolVar(&cfg.NoStatus, "nostatus", false, "Don't print status info (for if it messes with the terminal)")
	flag.Var(&cfg.Headers, "headers", "Additional headers to include with request. Supply as key:value. Can specify multiple - eg '-headers X-Forwarded-For:127.0.01 -headers X-ATT-DeviceId:XXXXX'")
	flag.StringVar(&cfg.Auth, "auth", "", "Basic auth. Supply this with the base64 encoded portion to be placed after the word 'Basic' in the Authorization header.")
	flag.BoolVar(&cfg.AppendDir, "appendSlash", false, "Append a / to all directory bruteforce requests (like extension, but slash instead of .yourthing)")

	flag.Parse()

	if showVersion {
		librecursebuster.PrintBanner(cfg)
		os.Exit(0)
	}

	printChan := make(chan librecursebuster.OutLine, 200)
	if cfg.URL == "" && cfg.InputList == "" { //&& cfg.InputList == ""
		flag.Usage()
		os.Exit(1)
	}
	var h *url.URL
	var err error
	if cfg.URL != "" {
		h, err = url.Parse(cfg.URL)
		if err != nil {
			panic("URL parse fail")
		}

		if h.Scheme == "" {
			if cfg.HTTPS {
				h, err = url.Parse("https://" + cfg.URL)
			} else {
				h, err = url.Parse("http://" + cfg.URL)
			}
		}

	}

	for _, x := range strings.Split(cfg.Extensions, ",") {
		state.Extensions = append(state.Extensions, x)
	}

	for _, x := range strings.Split(cfg.BadResponses, ",") {
		i, err := strconv.Atoi(x)
		if err != nil {
			panic(err)
		}
		state.BadResponses[i] = true
	}

	if cfg.Debug {
		go func() {
			http.ListenAndServe("localhost:6061", http.DefaultServeMux)
		}()
	}

	state.ParsedURL = h
	httpTransport := &http.Transport{MaxIdleConns: 100}
	client := &http.Client{Transport: httpTransport, Timeout: time.Duration(cfg.Timeout) * time.Second}

	if !cfg.FollowRedirects {
		client.CheckRedirect = librecursebuster.RedirectHandler
	}
	//skip ssl errors if requested to
	httpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.SSLIgnore}

	//setup channels
	pages := make(chan librecursebuster.SpiderPage, 1000)
	newPages := make(chan librecursebuster.SpiderPage, 10000)
	confirmed := make(chan librecursebuster.SpiderPage, 1000)
	workers := make(chan struct{}, cfg.Threads)
	maxDirs := make(chan struct{}, cfg.MaxDirs)
	testChan := make(chan string, 100)
	wg := &sync.WaitGroup{}

	state.Client = client

	//use a proxy if requested to
	if cfg.ProxyAddr != "" {

		if strings.HasPrefix(cfg.ProxyAddr, "http") {
			proxyUrl, err := url.Parse("http://127.0.0.1:8080")
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(proxyUrl)
			httpTransport.Proxy = http.ProxyURL(proxyUrl)

		} else {

			dialer, err := proxy.SOCKS5("tcp", cfg.ProxyAddr, nil, proxy.Direct)
			if err != nil {
				os.Exit(1)
			}
			httpTransport.Dial = dialer.Dial
		}
		librecursebuster.PrintOutput(fmt.Sprintf("Proxy set to: %s", cfg.ProxyAddr), librecursebuster.Info, 0, wg, printChan)

	}

	if cfg.BlacklistLocation != "" {
		readerChan := make(chan string, 100)
		go librecursebuster.LoadWords(cfg.BlacklistLocation, readerChan, printChan)
		for x := range readerChan {
			state.Blacklist[x] = true
		}
	}

	if cfg.WhitelistLocation != "" {
		readerChan := make(chan string, 100)
		go librecursebuster.LoadWords(cfg.WhitelistLocation, readerChan, printChan)
		for x := range readerChan {
			state.Whitelist[x] = true
		}
	}

	if cfg.Wordlist != "" {
		readerChan := make(chan string, 100)
		go librecursebuster.LoadWords(cfg.Wordlist, readerChan, printChan)
		for _ = range readerChan {
			state.WordlistLen++
		}
	}

	canary := librecursebuster.RandString(printChan)

	if cfg.Canary != "" {
		canary = cfg.Canary
	}

	librecursebuster.PrintBanner(cfg)
	prefix := h.String()
	if len(prefix) > 0 && string(prefix[len(prefix)-1]) != "/" {
		prefix = prefix + "/"
	}
	randURL := fmt.Sprintf("%s%s", prefix, canary)
	resp, x, err := librecursebuster.HttpReq("GET", randURL, client, cfg)
	if err != nil {
		panic("Canary Error, check url is correct: " + randURL + "\n" + err.Error())
	}
	librecursebuster.PrintOutput(
		fmt.Sprintf("Canary sent: %s, Response: %v", randURL, resp.Status),
		librecursebuster.Debug, 2, wg, printChan,
	)

	state.Soft404ResponseBody = x
	state.StartTime = time.Now()
	state.PerSecondShort = new(uint64)
	state.PerSecondLong = new(uint64)

	go librecursebuster.StatusPrinter(cfg, state, wg, printChan, testChan)
	go librecursebuster.ManageRequests(cfg, state, wg, pages, newPages, confirmed, workers, printChan, maxDirs, testChan)
	go librecursebuster.ManageNewURLs(cfg, state, wg, pages, newPages, printChan)
	go librecursebuster.OutputWriter(wg, cfg, confirmed, cfg.Localpath, printChan)
	go librecursebuster.StatsTracker(state)

	firstPage := librecursebuster.SpiderPage{}
	firstPage.Url = h.String()
	seedPages = append(seedPages, firstPage)

	librecursebuster.PrintOutput("Starting recursebuster...     ", librecursebuster.Info, 0, wg, printChan)

	//seed the workers
	for _, x := range seedPages {
		wg.Add(1)
		pages <- x
	}

	//wait for completion
	wg.Wait()

}
