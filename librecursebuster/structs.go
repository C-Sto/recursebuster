package librecursebuster

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/fatih/color"
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

//State represents the current state of the program. Options are not configured here, those are found in Config.
type State struct {
	//Should probably have different concepts between config and state. Configs that might change depending on the URL being queried

	//global State values
	Client         *http.Client
	TotalTested    *uint64
	PerSecondShort *uint64 //how many tested over 2 seconds or so
	PerSecondLong  *uint64
	StartTime      time.Time
	Blacklist      map[string]bool
	Whitelist      map[string]bool
	BadResponses   map[int]bool //response codes to consider *dont care* (this might be worth putting in per host state, but idk how)
	Extensions     []string
	Methods        []string
	WordlistLen    *uint32
	DirbProgress   *uint32

	//per host States
	Hosts HostStates
	//ParsedURL           *url.URL
	//Soft404ResponseBody []byte
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
func (hs *HostStates) AddSoft404Content(host string, content []byte) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.hosts[host] = HostState{ParsedURL: hs.hosts[host].ParsedURL, Soft404ResponseBody: content}
}

//Get404Body returns the stored 'not found' body
func (hs *HostStates) Get404Body(host string) []byte {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.hosts[host].Soft404ResponseBody
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
	Soft404ResponseBody []byte
}

//Config represents the configuration supplied at runtime. Different to program State which can change, this is set once, and only queried during program operation.
type Config struct {
	Agent             string
	AppendDir         bool
	Auth              string
	BadResponses      string
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
	MaxDirs           int
	Methods           string
	NoBase            bool
	NoGet             bool
	NoHead            bool
	NoRecursion       bool
	NoSpider          bool
	NoStatus          bool
	NoStartStop       bool
	NoWildcardChecks  bool
	ProxyAddr         string
	Ratio404          float64
	ShowAll           bool
	ShowLen           bool
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
	method    string
	Result    *http.Response
	Reference *url.URL //where did we get this URL from? (for the logic portion)
}
