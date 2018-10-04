package librecursebuster

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jroimartin/gocui"
)

//PrintBanner prints the banner and in debug mode will also print all set options
func PrintBanner(cfg *Config) {
	//todo: include settings in banner
	fmt.Println(strings.Repeat("=", 20))
	fmt.Println("recursebuster V" + cfg.Version)
	fmt.Println("Poorly hacked together by C_Sto (@C__Sto)")
	fmt.Println("Heavy influence from Gograbber, thx Swarlz")
	fmt.Println(strings.Repeat("=", 20))
	if cfg.Debug {
		printOpts(cfg)
		fmt.Println(strings.Repeat("=", 20))
	}
}

//stolen from swarlz
func printOpts(s *Config) {
	keys := reflect.ValueOf(&s).Elem()
	typeOfT := keys.Type()
	for i := 0; i < keys.NumField(); i++ {
		f := keys.Field(i)
		Debug.Printf("%s: = %v\n", typeOfT.Field(i).Name, f.Interface())
	}

}

//OutputWriter will write to a file and the screen
func OutputWriter(wg *sync.WaitGroup, cfg *Config, confirmed chan SpiderPage, localPath string, printChan chan OutLine) {
	//output worker
	pages := make(map[string]bool) //keep it unique
	file, err := os.OpenFile(localPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic("Can't open file for reading, is something wrong?\n" + err.Error())
	}
	defer file.Close()

	stringToWrite := "%s %s [%s]"
	stringToPrint := "Found %s %s [%s]"
	if cfg.ShowLen {
		stringToWrite = "%s %s [%s] Length: %v"
		stringToPrint = "%s Found %s [%s] Length: %v"
	}
	if cfg.CleanOutput {
		stringToWrite = "%s"
	}
	for {
		object := <-confirmed
		page := object.URL
		if _, ok := pages[page]; !ok {
			pages[page] = true
			writeS := fmt.Sprintf(stringToWrite, object.Result.Request.Method, page, object.Result.Status)
			printS := fmt.Sprintf(stringToPrint, object.Result.Request.Method, page, object.Result.Status)
			if cfg.ShowLen {
				writeS = fmt.Sprintf(stringToWrite, object.Result.Request.Method, page, object.Result.Status, object.Result.ContentLength)
				printS = fmt.Sprintf(stringToPrint, object.Result.Request.Method, page, object.Result.Status, object.Result.ContentLength)
			}
			if object.Result.StatusCode >= 300 && object.Result.StatusCode < 400 {
				//must be a 300ish redirect
				writeS += " " + object.Result.Header.Get("Location")
				printS += " " + object.Result.Header.Get("Location")
			}
			if cfg.CleanOutput {
				writeS = fmt.Sprintf(stringToWrite, page)
			}
			file.WriteString(writeS + "\n")
			file.Sync()

			printBasedOnStatus(object.Result.StatusCode, printS, wg, printChan)
		}
		wg.Done()
	}
}

func printBasedOnStatus(status int, printS string, wg *sync.WaitGroup, printChan chan OutLine) {
	x := status
	if 199 < x && x < 300 { //2xx
		PrintOutput(printS, Good2, 0, wg, printChan)
	} else if 299 < x && x < 400 { //3xx
		PrintOutput(printS, Good3, 0, wg, printChan)
	} else if 399 < x && x < 500 { //4xx
		PrintOutput(printS, Good4, 0, wg, printChan)
	} else if 499 < x && x < 600 { //5xx
		PrintOutput(printS, Good5, 0, wg, printChan)
	} else {
		PrintOutput(printS, Goodx, 0, wg, printChan)
	}
}

//PrintOutput used to send output to the screen
func PrintOutput(message string, writer *ConsoleWriter, verboseLevel int, wg *sync.WaitGroup, printChan chan OutLine) {
	wg.Add(1)
	printChan <- OutLine{
		Content: message,
		Type:    writer,
		Level:   verboseLevel,
	}
}

func UIPrinter(cfg *Config, state *State, wg *sync.WaitGroup, printChan chan OutLine, testChan chan string) {
	tick := time.NewTicker(time.Second * 2)
	testedURL := ""
	for {
		select {
		case o := <-printChan:
			//something to print
			//v.Write([]byte(o.Content + "\n"))
			if cfg.VerboseLevel >= o.Level {
				addToMainUI(state, o.Content)
			}
			//state.ui.Update()
			//fmt.Fprintln(v, o.Content+"\n")

		case <-tick.C:
			//time has elapsed the amount of time - it's been 2 seconds

		case t := <-testChan:
			//URL has been assessed
			testedURL = t
		}
		writeStatus(state, testedURL)
	}
}

func addToMainUI(state *State, s string) {
	state.ui.Update(func(g *gocui.Gui) error {
		v, err := g.View("Main")
		if err != nil {
			return err
			// handle error
		}
		//remove from top line if more lines than view height
		_, y := v.Size()
		if len(v.ViewBufferLines()) > y {
			newWrite := v.ViewBufferLines()[1:] //pop off last element, and append new line
			newWrite = append(newWrite, s)
			v.Clear()
			//fmt.Fprint(v, newWrite)
			for _, line := range newWrite {
				if line == "" {
					continue
				}
				fmt.Fprintln(v, strings.Trim(line, "\n"))
			}
		} else {
			fmt.Fprintln(v, s)
		}
		return nil
	})
}

func writeStatus(state *State, s string) {
	state.ui.Update(func(g *gocui.Gui) error {
		v, err := g.View("Status")
		if err != nil {
			return err
			// handle error
		}
		v.Clear()
		fmt.Fprintln(v, getStatus(state))
		sprint := fmt.Sprintf("[%.2f%%%%]%s", 100*float64(atomic.LoadUint32(state.DirbProgress))/float64(atomic.LoadUint32(state.WordlistLen)), s)
		fmt.Fprintln(v, sprint)
		return nil
	})
}

//StatusPrinter is the function that performs all the status printing logic
func StatusPrinter(cfg *Config, state *State, wg *sync.WaitGroup, printChan chan OutLine, testChan chan string) {
	tick := time.NewTicker(time.Second * 2)
	status := getStatus(state)
	spacesToClear := 0
	testedURL := ""
	for {
		select {
		case o := <-printChan:
			//shoudln't need to check for status here..
			//clear the line before printing anything
			if cfg.NoUI {
				fmt.Printf("\r%s\r", strings.Repeat(" ", spacesToClear))
			}

			if cfg.VerboseLevel >= o.Level {
				if cfg.NoUI {
					o.Type.Println(o.Content)
					//don't need to remember spaces to clear this line - this is newline suffixed
				} else {
					v, err := state.ui.View("Main")
					if err != nil {
						panic(err)
					}
					fmt.Fprintln(v, o.Content+"\n")
					//v.Write([]byte(o.Content))
					//o.Type.Fprintf(v, o.Content, nil...)
				}
			}
			wg.Done()

		case <-tick.C: //time has elapsed the amount of time - it's been 2 seconds
			status = getStatus(state)

		case t := <-testChan: //a URL has been assessed
			status = getStatus(state)
			testedURL = t
		}

		if !cfg.NoStatus {
			//assemble the status string
			sprint := fmt.Sprintf("%s"+black.Sprintf(">"), status)
			if cfg.MaxDirs == 1 && cfg.Wordlist != "" {
				//this is the grossest format string I ever did see
				sprint += fmt.Sprintf("[%.2f%%%%]%s", 100*float64(atomic.LoadUint32(state.DirbProgress))/float64(atomic.LoadUint32(state.WordlistLen)), testedURL)
			} else {
				sprint += fmt.Sprintf("%s", testedURL)
			}

			//flush the line
			//fmt.Printf("\r%s\r", strings.Repeat(" ", spacesToClear))

			//Status.Printf(sprint + "\r")
			v, err := state.ui.View("Main")
			if err != nil {
				panic(err)
			}
			fmt.Fprintln(v, sprint+"\n")
			//remember how many spaces we need to use to clear the line (21 for the date and time prefix)
			spacesToClear = len(sprint) + 21
		}

	}
}

func getStatus(s *State) string {

	return fmt.Sprintf("Tested: %d Speed(2s): %d/s Speed: %d/s",
		atomic.LoadUint64(s.TotalTested),
		atomic.LoadUint64(s.PerSecondShort),
		atomic.LoadUint64(s.PerSecondLong),
	)
}

//StatsTracker updates the stats every so often
func StatsTracker(state *State) {
	tick := time.NewTicker(time.Second * 2)
	testedBefore := atomic.LoadUint64(state.TotalTested)
	timeBefore := time.Now()
	for range tick.C {
		testedNow := atomic.LoadUint64(state.TotalTested)

		//calculate short average (tested since last tick)
		testedInPeriod := testedNow - testedBefore
		timeInPeriod := time.Since(timeBefore)
		testedPerSecond := float64(testedInPeriod) / float64(timeInPeriod.Seconds())
		atomic.StoreUint64(state.PerSecondShort, uint64(testedPerSecond))

		//calculate long average (tested per second since start)
		testedInPeriod = testedNow
		timeInPeriod = time.Since(state.StartTime)
		testedPerSecond = float64(testedInPeriod) / float64(timeInPeriod.Seconds())
		atomic.StoreUint64(state.PerSecondLong, uint64(testedPerSecond))

		testedBefore = testedNow
		timeBefore = time.Now()
	}
}
