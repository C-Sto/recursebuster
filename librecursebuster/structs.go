package librecursebuster

import (
	"io"
	"net/http"
	"net/url"

	"github.com/fatih/color"
)

var (
	Good    *ConsoleWriter //*log.Logger
	Info    *ConsoleWriter
	Warning *ConsoleWriter
	Debug   *ConsoleWriter
	Error   *ConsoleWriter
	Status  *ConsoleWriter
)

func InitLogger(
	goodHandle io.Writer,
	infoHandle io.Writer,
	debugHandle io.Writer,
	warningHandle io.Writer,
	statusHandle io.Writer,
	errorHandle io.Writer) {

	Good = ConsoleWriter{}.New(goodHandle, //color.New().New(goodHandle,
		g.Sprintf("GOOD: "))
	//log.Ldate|log.Ltime)

	Info = ConsoleWriter{}.New(infoHandle,
		b.Sprintf("INFO: "))
	//log.Ldate|log.Ltime)

	Debug = ConsoleWriter{}.New(debugHandle,
		y.Sprintf("DEBUG: "))
	//log.Ldate|log.Ltime)

	Warning = ConsoleWriter{}.New(warningHandle,
		m.Sprintf("WARNING: "))
	//log.Ldate|log.Ltime)

	Status = ConsoleWriter{}.New(statusHandle,
		c.Sprintf(">"))
	//log.Ldate|log.Ltime)

	Error = ConsoleWriter{}.New(errorHandle,
		r.Sprintf("ERROR: "))
	//log.Ldate|log.Ltime)
}

var g = color.New(color.FgGreen, color.Bold)
var y = color.New(color.FgYellow, color.Bold)
var r = color.New(color.FgRed, color.Bold)
var m = color.New(color.FgMagenta, color.Bold)
var b = color.New(color.FgBlue, color.Bold)
var c = color.New(color.FgCyan, color.Bold)

type OutLine struct {
	Content string
	//Type    *log.Logger
	Type *ConsoleWriter
}

//Should probably have different concepts between config and state. Configs that might change depending on the URL being queried
type State struct {
	ParsedURL           *url.URL
	Client              *http.Client
	TotalTested         *uint64
	Soft404ResponseBody []byte
	//Maps are reference types, so they are allways passed by reference.
	Blacklist    map[string]bool
	Whitelist    map[string]bool
	BadResponses map[int]bool
	Extensions   []string
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
}

type SpiderPage struct {
	Url    string
	Depth  int
	Result *http.Response
}
