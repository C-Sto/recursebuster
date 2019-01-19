package librecursebuster

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

//All the ConsoleWriter stuff can probably be abstracted into an interface :thinkingemoji:
var (
	Good2 *ConsoleWriter //*log.Logger
	Good3 *ConsoleWriter //*log.Logger
	Good4 *ConsoleWriter //*log.Logger
	Good5 *ConsoleWriter //*log.Logger
	Goodx *ConsoleWriter //*log.Logger

	Info   *ConsoleWriter
	Debug  *ConsoleWriter
	Error  *ConsoleWriter
	Status *ConsoleWriter
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

	Good2 = ConsoleWriter{}.New(good2Handle, g.Sprintf("GOOD: "))
	Good3 = ConsoleWriter{}.New(good3Handle, y.Sprintf("GOOD: "))
	Good4 = ConsoleWriter{}.New(good4Handle, c.Sprintf("GOOD: "))
	Good5 = ConsoleWriter{}.New(good5Handle, b.Sprintf("GOOD: "))
	Goodx = ConsoleWriter{}.New(goodxHandle, m.Sprintf("GOOD: "))

	//Good_2xx = Green
	//Good_3xx = Yellow
	//Good_4xx = Cyan
	//Good_5xx = Blue
	//Good_xxx = Magenta

	Info = ConsoleWriter{}.New(infoHandle,
		w.Sprintf("INFO: "))

	Debug = ConsoleWriter{}.New(debugHandle,
		y.Sprintf("DEBUG: "))

	Status = ConsoleWriter{}.New(statusHandle,
		black.Sprintf(">"))

	Error = ConsoleWriter{}.New(errorHandle,
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
	Type    *ConsoleWriter
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

//State represents the current state of the program. Options are not configured here, those are found in Config.
type State struct {
	//Should probably have different concepts between config and state. Configs that might change depending on the URL being queried

	//global State values
	Client         *http.Client
	BurpClient     *http.Client
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

	bodyContent string

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

	s.Client = s.ConfigureHTTPClient(false)
	s.BurpClient = s.ConfigureHTTPClient(true)

	s.Version = s.Cfg.Version

	if s.Cfg.BlacklistLocation != "" {
		readerChan := make(chan string, 100)
		go LoadWords(s.Cfg.BlacklistLocation, readerChan)
		for x := range readerChan {
			s.Blacklist[x] = true
		}
	}

	if s.Cfg.BodyContent != "" {
		readerChan := make(chan string, 100)
		go LoadWords(s.Cfg.BodyContent, readerChan)
		lines := []string{}
		for x := range readerChan {
			lines = append(lines, x)
		}
		s.bodyContent = strings.Join(lines, "\n")
	}

	if s.Cfg.WhitelistLocation != "" {
		readerChan := make(chan string, 100)
		go LoadWords(s.Cfg.WhitelistLocation, readerChan)
		for x := range readerChan {
			s.Whitelist[x] = true
		}
	}

	if s.Cfg.Wordlist != "" { // && s.Cfg.MaxDirs == 1 {

		zerod := uint32(0)
		s.DirbProgress = &zerod

		readerChan := make(chan string, 100)
		go LoadWords(s.Cfg.Wordlist, readerChan)
		for v := range readerChan {
			s.WordList = append(s.WordList, v)
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
