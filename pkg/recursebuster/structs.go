package recursebuster

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/c-sto/recursebuster/pkg/net"

	"github.com/c-sto/recursebuster/pkg/consolewriter"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

//All the ConsoleWriter stuff can probably be abstracted into an interface :thinkingemoji:
var (
	Good2 *consolewriter.ConsoleWriter //*log.Logger
	Good3 *consolewriter.ConsoleWriter //*log.Logger
	Good4 *consolewriter.ConsoleWriter //*log.Logger
	Good5 *consolewriter.ConsoleWriter //*log.Logger
	Goodx *consolewriter.ConsoleWriter //*log.Logger

	Info   *consolewriter.ConsoleWriter
	Debug  *consolewriter.ConsoleWriter
	Error  *consolewriter.ConsoleWriter
	Status *consolewriter.ConsoleWriter
)

//InitLogger initialises the output writer stuff
func InitLogger(
	good2Handle io.Writer,
	good3Handle io.Writer,
	good4Handle io.Writer,
	good5Handle io.Writer,
	goodxHandle io.Writer,
	infoHandle io.Writer,
	debugHandle io.Writer,
	warningHandle io.Writer,
	statusHandle io.Writer,
	errorHandle io.Writer) {

	Good2 = consolewriter.ConsoleWriter{}.New(good2Handle, g.Sprintf("GOOD: "))
	Good3 = consolewriter.ConsoleWriter{}.New(good3Handle, y.Sprintf("GOOD: "))
	Good4 = consolewriter.ConsoleWriter{}.New(good4Handle, c.Sprintf("GOOD: "))
	Good5 = consolewriter.ConsoleWriter{}.New(good5Handle, b.Sprintf("GOOD: "))
	Goodx = consolewriter.ConsoleWriter{}.New(goodxHandle, m.Sprintf("GOOD: "))

	//Good_2xx = Green
	//Good_3xx = Yellow
	//Good_4xx = Cyan
	//Good_5xx = Blue
	//Good_xxx = Magenta

	Info = consolewriter.ConsoleWriter{}.New(infoHandle,
		w.Sprintf("INFO: "))

	Debug = consolewriter.ConsoleWriter{}.New(debugHandle,
		y.Sprintf("DEBUG: "))

	Status = consolewriter.ConsoleWriter{}.New(statusHandle,
		black.Sprintf(">"))

	Error = consolewriter.ConsoleWriter{}.New(errorHandle,
		r.Sprintf("ERROR: "))
}

var black = color.New(color.FgBlack, color.Bold, color.BgWhite) //status arrow
var r = color.New(color.FgRed, color.Bold)                      //error
var g = color.New(color.FgGreen, color.Bold)                    //2xx *
var y = color.New(color.FgYellow, color.Bold)                   //3xx *
var b = color.New(color.FgBlue, color.Bold)                     //5xx *
var m = color.New(color.FgMagenta, color.Bold)                  //xxx *
var c = color.New(color.FgCyan, color.Bold)                     //4xx *
var w = color.New(color.FgWhite, color.Bold)                    //info *

//OutLine represents some form of console output. Should consist of the content to output, the type of output and the verbosity level.
type OutLine struct {
	Content string
	Level   int //Define the log/verbosity level. 0 is normal, 1 is higher verbosity etc
	Type    *consolewriter.ConsoleWriter
}

//var s *State

//SetState will assign the global state object
//func (s *State) SetState(s *State) {
//s = s
//}

//Wait will wait until all the relevant waitgroups have completed
func (s *State) Wait() {
	s.StartWG.Wait()
	s.wg.Wait()
	if s.ui != nil {
		s.wg.Add(1)
		s.ui.Update(func(*gocui.Gui) error { return gocui.ErrQuit })
		s.wg.Wait()
	}
	//close all the chans to avoid leaking routines during tests
	//	close(s.Chans.confirmedChan)
	//	close(s.Chans.newPagesChan)
	//	close(s.Chans.pagesChan)
	//	close(s.Chans.printChan)
	//	close(s.Chans.testChan)
	//	close(s.Chans.workersChan)
	//StopUI()
}

type workUnit struct {
	Method    string
	URLString string
	//Client    *http.Client
}

func (c chans) PrintChan() chan OutLine {
	return c.printChan
}

func (c chans) ConfirmedChan() chan SpiderPage {
	return c.confirmedChan
}

type chans struct {
	pagesChan,
	newPagesChan,
	confirmedChan chan SpiderPage

	workersChan     chan workUnit
	lessWorkersChan chan struct{}
	printChan       chan OutLine
	testChan        chan string
}

func (c *chans) GetWorkers() chan workUnit {
	return c.workersChan
}

func (chans) Init() *chans {
	return &chans{
		pagesChan:       make(chan SpiderPage, 1000),
		newPagesChan:    make(chan SpiderPage, 10000),
		confirmedChan:   make(chan SpiderPage, 1000),
		workersChan:     make(chan workUnit, 1000),
		lessWorkersChan: make(chan struct{}, 5), //too bad if you want to add more than 5 at a time ok
		//maxDirs := make(chan struct{}, cfg.MaxDirs),
		testChan:  make(chan string, 100),
		printChan: make(chan OutLine, 100),
	}
}

func (s *State) SetUI(g *gocui.Gui) {
	s.ui = g
}

//State represents the current state of the program. Options are not configured here, those are found in Config.
type State struct {
	//Should probably have different concepts between config and state. Configs that might change depending on the URL being queried

	//global State values
	Client     *http.Client
	BurpClient *http.Client

	requester *net.Requester

	Cfg            *Config
	TotalTested    *uint64
	PerSecondShort *uint64 //how many tested over 2 seconds or so
	PerSecondLong  *uint64
	workerCount    *uint32 //probably doesn't need to be async safe, but whatever
	StartTime      time.Time
	Blacklist      map[string]bool
	Whitelist      map[string]bool
	BadResponses   map[int]bool //response codes to consider *dont care* (this might be worth putting in per host state, but idk how)
	GoodResponses  map[int]bool //response codes to consider *only care*
	Extensions     []string
	Methods        []string
	//	WordlistLen    *uint32
	WordList     []string
	DirbProgress *uint32

	StopDir chan struct{} //should probably have all the chans in here

	Checked map[string]bool
	CMut    *sync.RWMutex

	StartWG *sync.WaitGroup
	wg      *sync.WaitGroup

	BodyContent string

	ui *gocui.Gui
	//per host States
	Hosts HostStates
	Chans *chans
	//ParsedURL           *url.URL
	//Soft404ResponseBody []byte
	Version string
}

//AddWG adds a single value to the state waitgroup
func (s *State) AddWG() {
	s.wg.Add(1)
}

//DoneWG does what you think it does
func (s *State) DoneWG() {
	s.wg.Done()
}

func (s *State) AddWorker(g *gocui.Gui, v *gocui.View) error {
	atomic.AddUint32(s.workerCount, 1)
	go s.StartTestWorker()
	return nil
}

//StartManagers is the function that starts all the management goroutines
func (gState *State) StartManagers() {
	if gState.Cfg.NoUI {
		gState.PrintBanner()
		go gState.StatusPrinter()
	} else {
		go gState.UIPrinter()
	}
	go gState.ManageRequests()
	go gState.ManageNewURLs()
	go gState.OutputWriter(gState.Cfg.Localpath)
	go gState.StatsTracker()
	for x := 0; x < gState.Cfg.Threads; x++ {
		go gState.StartTestWorker()
	}
}

//ManageRequests handles the request workers
func (gState *State) ManageRequests() {
	//manages net request workers
	for {
		page := <-gState.Chans.pagesChan
		if gState.Blacklist[page.URL] {
			gState.wg.Done()
			gState.PrintOutput(fmt.Sprintf("Not testing blacklisted URL: %s", page.URL), Info, 0)
			continue
		}
		for _, method := range gState.Methods {
			if page.Result == nil && !gState.Cfg.NoBase {
				gState.wg.Add(1)
				gState.Chans.workersChan <- workUnit{
					Method:    method,
					URLString: page.URL,
				}
			}
			if gState.Cfg.Wordlist != "" && string(page.URL[len(page.URL)-1]) == "/" { //if we are testing a directory
				gState.dirBust(page)
			}
		}
		gState.wg.Done()

	}
}

//ManageNewURLs will take in any URL, and decide if it should be added to the queue for bustin', or if we discovered something new
func (gState *State) ManageNewURLs() {
	//decides on whether to add to the directory list, or add to file output
	for {
		candidate := <-gState.Chans.newPagesChan
		//check the candidate is an actual URL
		//handle that one crazy case where :/ might be at the start because reasons
		if strings.HasPrefix(candidate.URL, "://") {
			//add a garbage scheme to get past the url parse stuff (the scheme will be added from the reference anyway)
			candidate.URL = "xxx" + candidate.URL
		}
		u, err := url.Parse(strings.TrimSpace(candidate.URL))

		if err != nil {
			gState.wg.Done()
			gState.PrintOutput(err.Error(), Error, 0)
			continue //probably a better way of doing this
		}

		//links of the form <a href="/thing" ></a> don't have a host portion to the URL
		if u.Host == "" {
			u.Host = candidate.Reference.Host
		}

		//actualUrl := gState.ParsedURL.Scheme + "://" + u.Host
		actualURL := net.CleanURL(u, (*candidate.Reference).Scheme+"://"+u.Host)

		gState.CMut.Lock()
		if _, ok := gState.Checked[actualURL]; !ok && //must have not checked it before
			(gState.Hosts.HostExists(u.Host) || gState.Whitelist[u.Host]) && //must be within whitelist, or be one of the starting urls
			!gState.Cfg.NoRecursion { //no recursion means we don't care about adding extra paths or content
			gState.Checked[actualURL] = true
			gState.CMut.Unlock()
			gState.wg.Add(1)
			gState.Chans.pagesChan <- SpiderPage{URL: actualURL, Reference: candidate.Reference, Result: candidate.Result}
			gState.PrintOutput("URL Added: "+actualURL, Debug, 3)

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
				newPage.Result = candidate.Result
				gState.CMut.RLock()
				if gState.Checked[newDir] {
					gState.CMut.RUnlock()
					continue
				}
				gState.CMut.RUnlock()
				gState.wg.Add(1)
				gState.Chans.newPagesChan <- newPage
			}
		} else {
			gState.CMut.Unlock()
		}

		gState.wg.Done()
	}
}

//RandString will return a UUID
func RandString() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

}

func (gState *State) dirBust(page SpiderPage) {
	//ugh
	u, err := url.Parse(page.URL)
	if err != nil {
		gState.PrintOutput("This should never occur, url parse error on parsed url?"+err.Error(), Error, 0)
		return
	}
	//check to make sure we aren't dirbusting a wildcardyboi (NOTE!!! USES FIRST SPECIFIED MEHTOD TO DO SOFT 404!)
	if !gState.Cfg.NoWildcardChecks {
		//gState.Chans.workersChan <- struct{}{}
		h, _, res := gState.evaluateURL(gState.Methods[0], page.URL+RandString(), gState.Client)
		//fmt.Println(page.URL, h, res)
		if res { //true response indicates a good response for a guid path, unlikely good
			is404, _ := net.DetectSoft404(h, gState.Hosts.Get404(u.Host), gState.Cfg.Ratio404)
			if is404 {
				//it's a soft404 probably, guess we can continue (this logic seems wrong??)
			} else {
				gState.PrintOutput(
					fmt.Sprintf("Wildcard response detected, skipping dirbusting of %s", page.URL),
					Info, 0)
				return
			}
		}
	}

	if !gState.Cfg.NoStartStop {
		gState.PrintOutput(
			fmt.Sprintf("Dirbusting %s", page.URL),
			Info, 0,
		)
	}

	atomic.StoreUint32(gState.DirbProgress, 0)
	//ensure we don't send things more than once
	for _, word := range gState.WordList { //will receive from the channel until it's closed
		if !gState.Cfg.NoEncode {
			word = url.PathEscape(word)
		}
		atomic.AddUint32(gState.DirbProgress, 1)
		//read words off the channel, and test it OR close out because we wanna skip it
		if word == "" {
			continue
		}
		select {
		case <-gState.StopDir:
			//<-maxDirs
			if !gState.Cfg.NoStartStop {
				gState.PrintOutput(fmt.Sprintf("Finished dirbusting: %s", page.URL), Info, 0)
			}
			return
		default:
			gState.combinate(page, word)
		}
	}
	//<-maxDirs
	if !gState.Cfg.NoStartStop {
		gState.PrintOutput(fmt.Sprintf("Finished dirbusting: %s", page.URL), Info, 0)
	}
}

func (gState *State) combinate(page SpiderPage, word string) {
	for _, method := range gState.Methods {
		if len(gState.Extensions) > 0 && gState.Extensions[0] != "" {
			for _, ext := range gState.Extensions {
				gState.CMut.Lock()
				if gState.Checked[method+page.URL+word+"."+ext] {
					gState.CMut.Unlock()
					continue
				}
				gState.Checked[method+page.URL+word+"."+ext] = true
				gState.CMut.Unlock()
				gState.wg.Add(1)
				gState.Chans.workersChan <- workUnit{
					Method:    method,
					URLString: page.URL + word + "." + ext,
				}
			}
		}
		if gState.Cfg.AppendDir {
			gState.CMut.Lock()
			if gState.Checked[method+page.URL+word+"/"] {
				gState.CMut.Unlock()
				continue
			}
			gState.Checked[method+page.URL+word+"/"] = true
			gState.CMut.Unlock()
			gState.wg.Add(1)
			gState.Chans.workersChan <- workUnit{
				Method:    method,
				URLString: page.URL + word + "/",
			}
		}
		gState.CMut.Lock()
		if gState.Checked[method+page.URL+word] {
			gState.CMut.Unlock()
			continue
		}
		gState.Checked[method+page.URL+word] = true
		gState.CMut.Unlock()
		gState.wg.Add(1)
		gState.Chans.workersChan <- workUnit{
			Method:    method,
			URLString: page.URL + word,
		}
		//if gState.Cfg.MaxDirs == 1 {
		//}
	}
}

//StartBusting will add a suppllied url to the queue to be tested
func (gState *State) StartBusting(randURL string, u url.URL) {
	if gState.requester == nil {
		gState.requester = net.NewRequester([]byte(gState.BodyContent),
			gState.Cfg.Agent, gState.Cfg.Cookies, gState.Cfg.Auth, gState.Cfg.Vhost, gState.Cfg.Headers, gState.Blacklist)

	}
	defer gState.wg.Done()
	if !gState.Cfg.NoWildcardChecks {
		resp, err := gState.requester.HTTPReq("GET", randURL, gState.Client)
		//<-gState.Chans.workersChan
		if err != nil {
			if gState.Cfg.InputList != "" {
				gState.PrintOutput(
					err.Error(),
					Error,
					0,
				)
				return
			}
			gState.PrintOutput("Canary Error, check url is correct: "+randURL+"\n"+err.Error(), Error, 1)
			return
		}
		gState.PrintOutput(
			fmt.Sprintf("Canary sent: %s, Response: %v", randURL, resp.Status),
			Debug, 2,
		)
		content, _ := ioutil.ReadAll(resp.Body)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(content))
		gState.Hosts.AddSoft404Content(u.Host, content, resp) // Soft404ResponseBody = xx
	} else {
		//<-gState.Chans.workersChan
	}
	x := SpiderPage{}
	x.URL = u.String()
	x.Reference = &u

	gState.CMut.Lock()
	defer gState.CMut.Unlock()
	if ok := gState.Checked[u.String()+"/"]; !strings.HasSuffix(u.String(), "/") && !ok {
		gState.wg.Add(1)
		gState.Chans.pagesChan <- SpiderPage{
			URL:       u.String() + "/",
			Reference: &u,
		}
		gState.Checked[u.String()+"/"] = true
		gState.PrintOutput("URL Added: "+u.String()+"/", Debug, 3)
	}
	if ok := gState.Checked[x.URL]; !ok {
		gState.wg.Add(1)
		gState.Chans.pagesChan <- x
		gState.Checked[x.URL] = true
		gState.PrintOutput("URL Added: "+x.URL, Debug, 3)
	}
	//got this far, check for robots and add that too
	if !gState.Cfg.NoRobots {
		gState.getRobots(u)
	}
}

func (gState *State) getRobots(u url.URL) {
	//make url good
	robotsUrl := u.String()
	if !strings.HasSuffix(robotsUrl, "/") {
		robotsUrl = robotsUrl + "/"
	}
	robotsUrl = robotsUrl + "robots.txt"
	resp, err := gState.requester.HTTPReq("GET", robotsUrl, gState.Client)
	//<-gState.Chans.workersChan
	if err != nil {
		if gState.Cfg.InputList != "" {
			gState.PrintOutput(
				err.Error(),
				Error,
				0,
			)
			return
		}
		gState.PrintOutput("Robots Error, check url is correct: "+robotsUrl+"\n"+err.Error(), Error, 1)
		return
	}
	gState.PrintOutput(
		fmt.Sprintf("Robots retreived: %s, Response: %v", robotsUrl, resp.Status),
		Debug, 2,
	)
	//https://github.com/samclarke/robotstxt/blob/master/robotstxt.go thx samclarke
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		gState.PrintOutput("Robots Error: \n"+err.Error(), Error, 1)
		return
	}
	// Check that the file is actually a plaintext robots.txt and not a soft404
	re, err := regexp.Compile(`(?i)<\s?html\s?>`)
	if err != nil {
		gState.PrintOutput("Robots Error: \n"+err.Error(), Error, 1)
		return
	}
	if re.Match(content) {
		// Soft404 or some other dodgy robots.txt discovered
		gState.PrintOutput("Robots Error: \n"+"Unexpected robots.txt content (contained html?)", Error, 1)
		return
	}

	//parse robots.txt

	contents := strings.Split(string(content), "\n")
	for _, line := range contents {
		//split into parts
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 1 {
			rule, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			switch strings.ToLower(rule) {
			case "user-agent":
				fallthrough
			case "crawl-delay":
				fallthrough
			case "host":
				//don't care
			case "sitemap":
				//don't care at the moment, this seems a logical thing to parse and add to the search though
			default:
				//everything else
				if val != "" {

					x := SpiderPage{}
					x.Reference = &u
					x.URL = u.String()

					if !strings.HasSuffix(x.URL, "/") && !strings.HasPrefix(val, "/") {
						x.URL = x.URL + "/"
					}
					x.URL = x.URL + val
					gState.wg.Add(1)
					gState.Chans.newPagesChan <- x

				}
			}
		}
	}
}

func (s *State) HandleX(g *gocui.Gui, v *gocui.View) error {
	select { //lol dope hack to stop it blocking
	case s.StopDir <- struct{}{}:
	default:
	}
	return nil
}

func (gState *State) StartTestWorker() {
	for {
		select {
		case w := <-gState.Chans.workersChan:
			gState.testURL(w.Method, w.URLString, gState.Client)
		case <-gState.Chans.lessWorkersChan:
			return
		}
	}
}

func (gState *State) testURL(method string, urlString string, client *http.Client) {
	defer func() {
		gState.wg.Done()
		atomic.AddUint64(gState.TotalTested, 1)
	}()
	select {
	case gState.Chans.testChan <- method + ":" + urlString:
	default: //this is to prevent blocking, it doesn't _really_ matter if it doesn't get written to output
	}
	headResp, content, good := gState.evaluateURL(method, urlString, client)

	if !good && !gState.Cfg.ShowAll {
		return
	}

	//inception (but also check if the directory version is good, and add that to the queue too)
	if string(urlString[len(urlString)-1]) != "/" && good {
		gState.wg.Add(1)
		gState.Chans.newPagesChan <- SpiderPage{URL: urlString + "/", Reference: headResp.Request.URL, Result: headResp}
	}

	gState.wg.Add(1)
	if headResp == nil {
		gState.Chans.confirmedChan <- SpiderPage{URL: urlString, Result: headResp, Reference: nil}
	} else {
		gState.Chans.confirmedChan <- SpiderPage{URL: urlString, Result: headResp, Reference: headResp.Request.URL}
	}
	if !gState.Cfg.NoSpider && good && !gState.Cfg.NoRecursion {
		urls, err := net.GetURLs(content)
		if err != nil {
			gState.PrintOutput(err.Error(), Error, 0)
		}
		for _, x := range urls { //add any found pages into the pool
			//add all the directories
			newPage := SpiderPage{}
			newPage.URL = x
			newPage.Reference = headResp.Request.URL

			gState.PrintOutput(
				fmt.Sprintf("Found URL on page: %s", x),
				Debug, 3,
			)

			gState.wg.Add(1)
			gState.Chans.newPagesChan <- newPage
		}
	}
}

func (s *State) StopWorker(g *gocui.Gui, v *gocui.View) error {
	count := atomic.LoadUint32(s.workerCount)
	if count == 0 { //avoid underflow
		return nil
	}
	count = count - 1
	atomic.StoreUint32(s.workerCount, count)
	s.Chans.lessWorkersChan <- struct{}{}
	return nil
}

//Init returns a new state value with initialised attributes
func (State) Init() *State {
	s := &State{
		BadResponses:   make(map[int]bool),
		GoodResponses:  make(map[int]bool),
		Whitelist:      make(map[string]bool),
		Blacklist:      make(map[string]bool),
		StopDir:        make(chan struct{}, 1),
		CMut:           &sync.RWMutex{},
		Checked:        make(map[string]bool),
		wg:             &sync.WaitGroup{},
		StartWG:        &sync.WaitGroup{},
		Chans:          chans{}.Init(),
		StartTime:      time.Now(),
		PerSecondShort: new(uint64),
		PerSecondLong:  new(uint64),
		TotalTested:    new(uint64),
		Cfg:            &Config{},
	}
	return s
}

//HostStates represents the interface to the Host States..? (all this smells of bad hacks)
type HostStates struct {
	mu    *sync.RWMutex
	hosts map[string]HostState
}

//Init initialises the map because apparently OO is hard to do
func (hs *HostStates) Init() {
	hs.mu = &sync.RWMutex{}
	hs.hosts = make(map[string]HostState)
}

//AddHost adds a host to the hosts lol
func (hs *HostStates) AddHost(u *url.URL) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.hosts[u.Host] = HostState{ParsedURL: u}
}

//AddSoft404Content sets the soft404 content retreived using the canary request to be compared against during the hacking phase
func (hs *HostStates) AddSoft404Content(host string, content []byte, h *http.Response) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.hosts[host] = HostState{ParsedURL: hs.hosts[host].ParsedURL, Soft404ResponseBody: content, Response404: h}
}

//Get404Body returns the stored known-not-good body from a response
func (hs *HostStates) Get404Body(host string) []byte {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.hosts[host].Soft404ResponseBody
}

//Get404 returns the stored known-not-good response
func (hs *HostStates) Get404(host string) *http.Response {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.hosts[host].Response404
}

//HostExists checks to see if the host string specified exists within the hosts states??
func (hs HostStates) HostExists(hval string) bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	_, ok := hs.hosts[hval]
	return ok
}

//HostState is the actual state of each host (wow this is confusing and should be broken into different state files imo)
type HostState struct {
	ParsedURL           *url.URL
	Response404         *http.Response
	Soft404ResponseBody []byte
}

//Config represents the configuration supplied at runtime. Different to program State which can change, this is set once, and only queried during program operation.
type Config struct {
	Agent             string
	Ajax              bool
	AppendDir         bool
	Auth              string
	BadResponses      string
	BadBod            string
	GoodResponses     string
	BadHeader         ArrayStringFlag
	BodyContent       string
	BlacklistLocation string
	BurpMode          bool
	Canary            string
	CleanOutput       bool
	Cookies           string
	Debug             bool
	Extensions        string
	FollowRedirects   bool
	Headers           ArrayStringFlag
	HTTPS             bool
	InputList         string
	Localpath         string
	//MaxDirs           int
	Methods           string
	NoBase            bool
	NoGet             bool
	NoEncode          bool
	NoHead            bool
	NoRecursion       bool
	NoRobots          bool
	NoSpider          bool
	NoStatus          bool
	NoStartStop       bool
	NoUI              bool
	NoWildcardChecks  bool
	ProxyAddr         string
	Ratio404          float64
	ShowAll           bool
	ShowLen           bool
	ShowVersion       bool
	SSLIgnore         bool
	Threads           int
	Timeout           int
	URL               string
	VerboseLevel      int
	Version           string
	Vhost             string
	WhitelistLocation string
	Wordlist          string
}

//ArrayStringFlag is used to be able to supply more than one flag at the command line (for the -header option)
type ArrayStringFlag []string

func (i *ArrayStringFlag) String() string {
	return fmt.Sprintf("%v", *i)
}

//Set the value. Appends to the current state. (required for the interface)
func (i *ArrayStringFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

//Get the Values stored (required for the interface)
func (i *ArrayStringFlag) Get() []string {
	return *i
}

//SpiderPage represents a 'working' page object, represented by an URL and it's (optional)result.
type SpiderPage struct {
	URL       string
	Result    *http.Response
	Reference *url.URL //where did we get this URL from? (for the logic portion)
}

//SetupState will perform all the basic state setup functions (adding URL's to the blacklist etc)
func (s *State) SetupState() {

	//set workers (whoops)
	//s.Chans.workersChan = make(chan struct{}, s.Cfg.Threads)

	if s.Cfg.Ajax {
		s.Cfg.Headers = append(s.Cfg.Headers, "X-Requested-With:XMLHttpRequest")
	}

	for _, x := range strings.Split(s.Cfg.Extensions, ",") {
		s.Extensions = append(s.Extensions, x)
	}

	for _, x := range strings.Split(s.Cfg.Methods, ",") {
		s.Methods = append(s.Methods, x)
	}

	for _, x := range strings.Split(s.Cfg.BadResponses, ",") {
		i, err := strconv.Atoi(x)
		if err != nil {
			fmt.Println("Bad response code supplied!")
			panic(err)
		}
		s.BadResponses[i] = true //this is probably a candidate for individual urls. Unsure how to config that cleanly though
	}

	for _, x := range strings.Split(s.Cfg.GoodResponses, ",") {
		if x == "" {
			continue
		}
		i, err := strconv.Atoi(x)
		if err != nil {
			fmt.Println("Bad response code supplied!")
			panic(err)
		}
		s.GoodResponses[i] = true
	}

	s.Client = net.ConfigureHTTPClient(s.Cfg.ProxyAddr, s.Cfg.Timeout, false, s.Cfg.BurpMode, s.Cfg.FollowRedirects, s.Cfg.SSLIgnore)
	s.BurpClient = net.ConfigureHTTPClient(s.Cfg.ProxyAddr, s.Cfg.Timeout, true, s.Cfg.BurpMode, s.Cfg.FollowRedirects, s.Cfg.SSLIgnore)

	s.Version = s.Cfg.Version

	if s.Cfg.BlacklistLocation != "" {
		b := LoadWords(s.Cfg.BlacklistLocation)
		for b.Scan() {
			s.Blacklist[b.Text()] = true
		}
	}

	if s.Cfg.BodyContent != "" {
		b := LoadWords(s.Cfg.BodyContent)
		lines := []string{}
		for b.Scan() {
			lines = append(lines, b.Text())
		}
		s.BodyContent = strings.Join(lines, "\n")
	}

	if s.Cfg.WhitelistLocation != "" {
		b := LoadWords(s.Cfg.WhitelistLocation)
		for b.Scan() {
			s.Whitelist[b.Text()] = true
		}
	}

	if s.Cfg.Wordlist != "" { // && s.Cfg.MaxDirs == 1 {

		zerod := uint32(0)
		s.DirbProgress = &zerod

		b := LoadWords(s.Cfg.Wordlist)
		//readerChan := make(chan string, 100)
		//go LoadWords(s.Cfg.Wordlist, readerChan)
		for b.Scan() {
			s.WordList = append(s.WordList, b.Text())
			//atomic.AddUint32(s.WordlistLen, 1)
		}
	}
	workers := uint32(s.Cfg.Threads)
	s.workerCount = &workers
	s.StartTime = time.Now()
	s.PerSecondShort = new(uint64)
	s.PerSecondLong = new(uint64)

}
func getURLSlice(globalState *State) []string {
	urlSlice := []string{}
	if globalState.Cfg.URL != "" {
		urlSlice = append(urlSlice, globalState.Cfg.URL)
	}
	return urlSlice
}

func LoadWords(filename string) *bufio.Scanner {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	return bufio.NewScanner(f)
}
