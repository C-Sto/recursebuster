package librecursebuster

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jroimartin/gocui"
)

//PrintBanner prints the banner and in debug mode will also print all set options
func PrintBanner(cfg *Config) {
	//todo: include settings in banner
	if cfg.NoUI || cfg.ShowVersion {
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
}

//stolen from swarlz
func printOpts(s *Config) {
	keys := reflect.ValueOf(s).Elem()
	typeOfT := keys.Type()
	for i := 0; i < keys.NumField(); i++ {
		f := keys.Field(i)
		Debug.Printf("%s: = %v\n", typeOfT.Field(i).Name, f.Interface())
	}

}

//OutputWriter will write to a file and the screen
func OutputWriter(cfg *Config, localPath string) {
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
		object := <-gState.Chans.confirmedChan
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

			printBasedOnStatus(object.Result.StatusCode, printS)
		}
		gState.wg.Done()
		//wg.Done()
	}
}

func printBasedOnStatus(status int, printS string) {
	x := status
	if 199 < x && x < 300 { //2xx
		PrintOutput(printS, Good2, 0)
	} else if 299 < x && x < 400 { //3xx
		PrintOutput(printS, Good3, 0)
	} else if 399 < x && x < 500 { //4xx
		PrintOutput(printS, Good4, 0)
	} else if 499 < x && x < 600 { //5xx
		PrintOutput(printS, Good5, 0)
	} else {
		PrintOutput(printS, Goodx, 0)
	}
}

//PrintOutput used to send output to the screen
func PrintOutput(message string, writer *ConsoleWriter, verboseLevel int) {
	gState.wg.Add(1)
	gState.Chans.printChan <- OutLine{
		Content: message,
		Type:    writer,
		Level:   verboseLevel,
	}
}

//UIPrinter is called to write a pretty UI
func UIPrinter(cfg *Config) {
	tick := time.NewTicker(time.Second / 30) //30 'fps'
	testedURL := ""
	for {
		select {
		case o := <-gState.Chans.printChan:
			//something to print
			//v.Write([]byte(o.Content + "\n"))
			if cfg.VerboseLevel >= o.Level {
				addToMainUI(o)
			}
			gState.wg.Done()
			//gState.ui.Update()
			//fmt.Fprintln(v, o.Content+"\n")

		case <-tick.C:
			writeStatus(testedURL)
			//refreshUI() //time has elapsed the amount of time - it's been 2 seconds
			updateUI()

		case t := <-gState.Chans.testChan:
			//URL has been assessed
			testedURL = t
		}

	}
}

func updateUI() {
	gState.ui.Update(func(g *gocui.Gui) error { return nil })

}

func addToMainUI(o OutLine) { //s string) {
	//go gState.ui.Update(func(g *gocui.Gui) error {
	//g := gState.ui
	v, err := gState.ui.View("Main")
	if err != nil {
		return // err
	}
	fmt.Fprintln(v, o.Type.GetPrefix()+o.Content)
	return // nil
	//})
}

func writeStatus(s string) {
	//go gState.ui.Update(func(g *gocui.Gui) error {
	//g := gState.ui
	v, err := gState.ui.View("Status")
	if err != nil {
		return //err
		// handle error
	}
	v.Clear()
	fmt.Fprintln(v, getStatus())
	sprint := ""
	if len(gState.WordList) > 0 {
		sprint = fmt.Sprintf("[%.2f%%%%]%s", 100*float64(atomic.LoadUint32(gState.DirbProgress))/float64(len(gState.WordList)), s)
	} else {
		sprint = fmt.Sprintf("Waiting on %v items", gState.wg)
	}
	fmt.Fprintln(v, sprint)
	fmt.Fprintln(v, "ctrl + [(c) quit, (x) stop current dir], (arrow up/down) move one line, (pgup/pgdown) move 10 lines")
	fmt.Fprintln(v, time.Now().String())
	return //nil
	//})
}

//StatusPrinter is the function that performs all the status printing logic
func StatusPrinter(cfg *Config) {
	tick := time.NewTicker(time.Second * 2)
	status := getStatus()
	spacesToClear := 0
	testedURL := ""
	for {
		select {
		case o := <-gState.Chans.printChan:
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
					v, err := gState.ui.View("Main")
					if err != nil {
						panic(err)
					}
					fmt.Fprintln(v, o.Content+"\n")
				}
			}
			gState.wg.Done()

		case <-tick.C: //time has elapsed the amount of time - it's been 2 seconds
			status = getStatus()

		case t := <-gState.Chans.testChan: //a URL has been assessed
			status = getStatus()
			testedURL = t
		}

		if !cfg.NoStatus && cfg.NoUI {
			//assemble the status string
			sprint := fmt.Sprintf("%s"+black.Sprint(">"), status)
			//if cfg.MaxDirs == 1 && cfg.Wordlist != "" {
			//this is the grossest format string I ever did see
			if len(gState.WordList) > 0 {
				sprint += fmt.Sprintf("[%.2f%%%%]%s", 100*float64(atomic.LoadUint32(gState.DirbProgress))/float64(len(gState.WordList)), testedURL)
			}
			//} else {
			//	sprint += fmt.Sprintf("%s", testedURL)
			//}

			//flush the line
			fmt.Printf("\r%s\r", strings.Repeat(" ", spacesToClear))

			Status.Printf(sprint + "\r")
			/*		v, err := gState.ui.View("Main")
					if err != nil {
						panic(err)
					}
					fmt.Fprintln(v, sprint+"\n")*/
			//remember how many spaces we need to use to clear the line (21 for the date and time prefix)
			spacesToClear = len(sprint) + 21
		}

	}
}

func getStatus() string {
	return fmt.Sprintf("Tested: %d Speed(2s): %d/s Speed: %d/s",
		atomic.LoadUint64(gState.TotalTested),
		atomic.LoadUint64(gState.PerSecondShort),
		atomic.LoadUint64(gState.PerSecondLong),
	)
}

//StatsTracker updates the stats every so often
func StatsTracker() {
	tick := time.NewTicker(time.Second * 2)
	testedBefore := atomic.LoadUint64(gState.TotalTested)
	timeBefore := time.Now()
	for range tick.C {
		testedNow := atomic.LoadUint64(gState.TotalTested)

		//calculate short average (tested since last tick)
		testedInPeriod := testedNow - testedBefore
		timeInPeriod := time.Since(timeBefore)
		testedPerSecond := float64(testedInPeriod) / float64(timeInPeriod.Seconds())
		atomic.StoreUint64(gState.PerSecondShort, uint64(testedPerSecond))

		//calculate long average (tested per second since start)
		testedInPeriod = testedNow
		timeInPeriod = time.Since(gState.StartTime)
		testedPerSecond = float64(testedInPeriod) / float64(timeInPeriod.Seconds())
		atomic.StoreUint64(gState.PerSecondLong, uint64(testedPerSecond))

		testedBefore = testedNow
		timeBefore = time.Now()
	}
}
