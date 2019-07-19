package cli

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/c-sto/recursebuster/pkg/ui"

	"github.com/c-sto/recursebuster/pkg/recursebuster"
)

//Run starts a new recursebuster instance to do the thing. Assumes state has been configured
func Run(globalState *recursebuster.State) {

	if globalState.Cfg.ShowVersion {
		globalState.PrintBanner()
		os.Exit(0)
	}

	if globalState.Cfg.URL == "" && globalState.Cfg.InputList == "" {
		flag.Usage()
		os.Exit(1)
	}

	urlSlice := GetURLSlice(globalState)

	setupConfig(globalState, urlSlice[0])

	globalState.SetupState()

	//do first load of urls (send canary requests to make sure we can dirbust them)
	quitChan := make(chan struct{})
	if !globalState.Cfg.NoUI {
		uiWG := &sync.WaitGroup{}
		uiWG.Add(1)
		go uiQuit(quitChan)
		go ui.StartUI(uiWG, quitChan, globalState)
		uiWG.Wait()
	}

	globalState.StartManagers()

	globalState.PrintOutput("Starting recursebuster...", recursebuster.Info, 0)

	//seed the workers
	for _, s := range urlSlice {
		u, err := url.Parse(s)
		if err != nil {
			panic(err)
		}

		if u.Scheme == "" {
			if globalState.Cfg.HTTPS {
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
		randURL := fmt.Sprintf("%s%s", prefix, globalState.Cfg.Canary)
		//globalState.Chans.GetWorkers() <- struct{}{}
		globalState.AddWG()
		go globalState.StartBusting(randURL, *u)

	}

	//wait for completion
	globalState.Wait()

}

func GetURLSlice(globalState *recursebuster.State) []string {
	urlSlice := []string{}
	if globalState.Cfg.URL != "" {
		urlSlice = append(urlSlice, globalState.Cfg.URL)
	}

	if globalState.Cfg.InputList != "" { //can have both -u flag and -iL flag
		//must be using an input list
		f, err := os.Open(globalState.Cfg.InputList)
		if err != nil {
			panic(err)
		}
		b := bufio.NewScanner(f)

		for b.Scan() {
			//ensure all urls will parse good
			x := b.Text()
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

func setupConfig(globalState *recursebuster.State, urlSliceZero string) {
	if globalState.Cfg.Debug {
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
		if globalState.Cfg.HTTPS {
			h, err = url.Parse("https://" + urlSliceZero)
		} else {
			h, err = url.Parse("http://" + urlSliceZero)
		}
	}
	if err != nil {
		panic(err)
	}
	globalState.Hosts.AddHost(h)

	if globalState.Cfg.Canary == "" {
		globalState.Cfg.Canary = recursebuster.RandString()
	}

}
