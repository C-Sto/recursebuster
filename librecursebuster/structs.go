package librecursebuster

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/fatih/color"
)

var (
	Good2 *ConsoleWriter //*log.Logger
	Good3 *ConsoleWriter //*log.Logger
	Good4 *ConsoleWriter //*log.Logger
	Good5 *ConsoleWriter //*log.Logger
	Goodx *ConsoleWriter //*log.Logger

	Info    *ConsoleWriter
	Warning *ConsoleWriter
	Debug   *ConsoleWriter
	Error   *ConsoleWriter
	Status  *ConsoleWriter
)

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

	Good2 = ConsoleWriter{}.New(good2Handle, //color.New().New(goodHandle,
		g.Sprintf("GOOD: "))
	Good3 = ConsoleWriter{}.New(good3Handle, //color.New().New(goodHandle,
		y.Sprintf("GOOD: "))
	Good4 = ConsoleWriter{}.New(good4Handle, //color.New().New(goodHandle,
		c.Sprintf("GOOD: "))
	Good5 = ConsoleWriter{}.New(good5Handle, //color.New().New(goodHandle,
		b.Sprintf("GOOD: "))
	Goodx = ConsoleWriter{}.New(goodxHandle, //color.New().New(goodHandle,
		m.Sprintf("GOOD: "))

	//Good_2xx = Green
	//Good_3xx = Yellow
	//Good_4xx = Cyan
	//Good_5xx = Blue
	//Good_xxx = Magenta

	//log.Ldate|log.Ltime)

	Info = ConsoleWriter{}.New(infoHandle,
		w.Sprintf("INFO: "))
	//log.Ldate|log.Ltime)

	Debug = ConsoleWriter{}.New(debugHandle,
		y.Sprintf("DEBUG: "))
	//log.Ldate|log.Ltime)

	Status = ConsoleWriter{}.New(statusHandle,
		black.Sprintf(">"))
	//log.Ldate|log.Ltime)

	Error = ConsoleWriter{}.New(errorHandle,
		r.Sprintf("ERROR: "))
	//log.Ldate|log.Ltime)
}

var black = color.New(color.FgBlack, color.Bold, color.BgWhite) //status arrow
var r = color.New(color.FgRed, color.Bold)                      //error
var g = color.New(color.FgGreen, color.Bold)                    //2xx *
var y = color.New(color.FgYellow, color.Bold)                   //3xx *
var b = color.New(color.FgBlue, color.Bold)                     //5xx *
var m = color.New(color.FgMagenta, color.Bold)                  //xxx *
var c = color.New(color.FgCyan, color.Bold)                     //4xx *
var w = color.New(color.FgWhite, color.Bold)                    //info *

type OutLine struct {
	Content string
	Level   int //Define the log/verbosity level. 0 is normal, 1 is higher verbosity etc
	Type    *ConsoleWriter
}

//Should probably have different concepts between config and state. Configs that might change depending on the URL being queried
type State struct {
	ParsedURL           *url.URL
	Client              *http.Client
	TotalTested         *uint64
	PerSecondShort      *uint64
	PerSecondLong       *uint64
	Soft404ResponseBody []byte
	StartTime           time.Time
	//Maps are reference types, so they are allways passed by reference.
	Blacklist    map[string]bool
	Whitelist    map[string]bool
	BadResponses map[int]bool
	Extensions   []string
	WordlistLen  *uint32
	DirbProgress *uint32
}

//Things that will be global, regardless of what URL is being queried
type Config struct {
	Version           string
	Threads           int
	URL               string
	Localpath         string
	ProxyAddr         string
	SSLIgnore         bool
	Wordlist          string
	Canary            string
	Agent             string
	Ratio404          float64
	BlacklistLocation string
	WhitelistLocation string
	NoSpider          bool
	Timeout           int
	Debug             bool
	NoGet             bool
	MaxDirs           int
	ShowAll           bool
	ShowLen           bool
	FollowRedirects   bool
	BadResponses      string
	CleanOutput       bool
	Cookies           string
	Extensions        string
	InputList         string
	HTTPS             bool
	VerboseLevel      int
	NoStatus          bool
	Headers           ArrayStringFlag
	Auth              string
	AppendDir         bool
	NoRecursion       bool
	BurpMode          bool
}

type ArrayStringFlag []string

func (i *ArrayStringFlag) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *ArrayStringFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *ArrayStringFlag) Get() []string {
	return *i
}

type SpiderPage struct {
	Url    string
	Result *http.Response
}
