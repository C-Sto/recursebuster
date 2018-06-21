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

	//"./librecursebuster"
	"github.com/c-sto/recursebuster/librecursebuster"
	"github.com/fatih/color"
	"golang.org/x/net/proxy"
)

const version = "1.0.0"

func main() {
	if runtime.GOOS == "windows" { //lol goos
		//can't use color.Error, because *nix etc don't have that for some reason :(
		librecursebuster.InitLogger(color.Output, color.Output, color.Output, color.Output, color.Output, color.Output)
	} else {
		librecursebuster.InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	}
	cfg := librecursebuster.Config{}
	state := librecursebuster.State{
		BadResponses: make(map[int]bool),
		Whitelist:    make(map[string]bool),
		Blacklist:    make(map[string]bool),
	}

	cfg.Version = version
	totesTested := uint64(0)
	state.TotalTested = &totesTested
	flag.IntVar(&cfg.Threads, "t", 1, "Number of concurrent threads")
	flag.StringVar(&cfg.URL, "u", "", "Url to spider")
	flag.StringVar(&cfg.Localpath, "o", "."+string(os.PathSeparator)+"busted.txt", "Local file to dump into")
	flag.BoolVar(&cfg.SSLIgnore, "k", false, "Ignore SSL check")
	flag.StringVar(&cfg.ProxyAddr, "p", "", "Proxy configuration options in the form ip:port eg: 127.0.0.1:9050")
	flag.StringVar(&cfg.Wordlist, "w", "", "Wordlist to use for bruteforce. Blank for spider only")
	flag.StringVar(&cfg.Canary, "canary", "", "Custom value to use to check for wildcards")
	flag.StringVar(&cfg.Agent, "ua", "RecurseBuster/"+version, "User agent to use when sending requests.")
	flag.Float64Var(&cfg.Ratio404, "ratio", 0.95, "Similarity ratio to the 404 canary page.")
	flag.StringVar(&cfg.BlacklistLocation, "blacklist", "", "Blacklist of prefixes to not check. Will not check on exact matches.")
	flag.BoolVar(&cfg.NoSpider, "spider", false, "Search the page body for links, and directories to add to the spider queue.")
	flag.IntVar(&cfg.Timeout, "timeout", 20, "Timeout (seconds) for HTTP/TCP connections")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debugging")
	flag.BoolVar(&cfg.NoGet, "noget", false, "Do not perform a GET request (only use HEAD request/response)")
	flag.IntVar(&cfg.MaxDirs, "dirs", 1, "Maximum directories to perform busting on concurrently NOTE: directories will still be brute forced, this setting simply directs how many should be concurrently bruteforced")
	flag.BoolVar(&cfg.ShowAll, "all", false, "Show and write result of all checks")
	flag.BoolVar(&cfg.ShowLen, "len", false, "Show and write the length of the response")
	flag.StringVar(&cfg.WhitelistLocation, "whitelist", "", "Whitelist of domains to include in brute-force")
	flag.BoolVar(&cfg.FollowRedirects, "redirect", false, "Follow redirects")
	flag.StringVar(&cfg.BadResponses, "bad", "404", "Responses to consider 'bad' or 'not found'. Comma-separated This works the opposite way of gobuster!")
	flag.BoolVar(&cfg.CleanOutput, "clean", false, "Output clean urls to output file for easy loading into other tools and whatnot.")
	flag.StringVar(&cfg.Cookies, "cookies", "", "Any cookies to include with requests. This is smashed into the cookies header, so copy straight from burp I guess.")
	flag.StringVar(&cfg.Extensions, "ext", "", "Extensions to append to checks. Multiple extensions can be specified, comma separate them.")

	flag.Parse()

	if cfg.URL == "" { //&& cfg.InputList == ""
		flag.Usage()
		os.Exit(1)
	}

	h, err := url.Parse(cfg.URL)
	if err != nil {
		panic("url parse fail")
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
	printChan := make(chan librecursebuster.OutLine, 200)
	maxDirs := make(chan struct{}, cfg.MaxDirs)

	state.Client = client
	//user a proxy if requested to
	if cfg.ProxyAddr != "" {
		printChan <- librecursebuster.OutLine{Content: fmt.Sprintf("Proxy set to: %s", cfg.ProxyAddr), Type: librecursebuster.Info}

		dialer, err := proxy.SOCKS5("tcp", cfg.ProxyAddr, nil, proxy.Direct)
		if err != nil {
			os.Exit(1)
		}
		httpTransport.Dial = dialer.Dial
	}

	firstPage := librecursebuster.SpiderPage{}
	firstPage.Url = cfg.URL
	firstPage.Depth = 0

	wg := &sync.WaitGroup{}

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

	canary := librecursebuster.RandString(printChan)

	if cfg.Canary != "" {
		canary = cfg.Canary
	}

	librecursebuster.PrintBanner(cfg)

	prefix := cfg.URL
	if len(prefix) > 0 && string(prefix[len(prefix)-1]) != "/" {
		prefix = prefix + "/"
	}
	randURL := fmt.Sprintf("%s%s", prefix, canary)
	_, x, err := librecursebuster.HttpReq("GET", randURL, client, cfg)
	if err != nil {
		panic("Canary Error, check url is correct: " + randURL)
	}
	state.Soft404ResponseBody = x

	wg.Add(1)
	pages <- firstPage
	printChan <- librecursebuster.OutLine{Content: "Starting...", Type: librecursebuster.Info}
	//fmt.Println("Starting buster..")

	go librecursebuster.StatusPrinter(state.TotalTested, printChan)
	go librecursebuster.ManageRequests(cfg, state, wg, pages, newPages, confirmed, workers, printChan, maxDirs)
	go librecursebuster.ManageNewURLs(cfg, state, wg, pages, newPages, printChan)
	go librecursebuster.OutputWriter(wg, cfg, confirmed, cfg.Localpath, printChan)

	wg.Wait()

}
