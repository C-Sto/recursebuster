package librecursebuster

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func PrintBanner(cfg Config) {
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
func printOpts(s Config) {
	keys := reflect.ValueOf(&s).Elem()
	typeOfT := keys.Type()
	for i := 0; i < keys.NumField(); i++ {
		f := keys.Field(i)
		Debug.Printf("%s: = %v\n", typeOfT.Field(i).Name, f.Interface())
	}

}

func OutputWriter(wg *sync.WaitGroup, cfg Config, confirmed chan SpiderPage, localPath string, printChan chan OutLine) {
	//output worker
	pages := make(map[string]bool) //keep it unique
	file, err := os.OpenFile(localPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic("Can't open file for reading, is something wrong?\n" + err.Error())
	}
	defer file.Close()

	stringToWrite := "%s [%s]"
	stringToPrint := "Found %s [%s]"
	if cfg.ShowLen {
		stringToWrite = "%s [%s] Length: %v"
		stringToPrint = "Found %s [%s] Length: %v"
	}
	if cfg.CleanOutput {
		stringToWrite = "%s\n"
	}
	for {
		object := <-confirmed
		page := object.Url
		if _, ok := pages[page]; !ok {
			pages[page] = true
			writeS := fmt.Sprintf(stringToWrite, page, object.Result.Status)
			printS := fmt.Sprintf(stringToPrint, page, object.Result.Status)
			if cfg.ShowLen {
				writeS = fmt.Sprintf(stringToWrite, page, object.Result.Status, object.Result.ContentLength)
				printS = fmt.Sprintf(stringToPrint, page, object.Result.Status, object.Result.ContentLength)
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
			//ugh
			x := object.Result.StatusCode
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
		wg.Done()
	}
}

func PrintOutput(message string, writer *ConsoleWriter, verboseLevel int, wg *sync.WaitGroup, printChan chan OutLine) {
	wg.Add(1)
	printChan <- OutLine{
		Content: message,
		Type:    writer,
		Level:   verboseLevel,
	}
}

func StatusPrinter(cfg Config, state State, wg *sync.WaitGroup, printChan chan OutLine, testChan chan string) {
	tick := time.NewTicker(time.Second * 2)
	status := getStatus(state)
	spacesToClear := 0
	testedURL := ""
	for {
		select {
		case o := <-printChan:
			//shoudln't need to check for status here..

			//clear the line before printing anything
			fmt.Printf("\r%s\r", strings.Repeat(" ", spacesToClear))

			if cfg.VerboseLevel >= o.Level {
				o.Type.Println(o.Content)
				//don't need to remember spaces to clear this line - this is newline suffixed
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
			sprint := fmt.Sprintf("%s"+black.Sprintf(">")+"%s", status, testedURL)

			//flush the line
			fmt.Printf("\r%s\r", strings.Repeat(" ", spacesToClear))

			Status.Printf(sprint + "\r")
			//remember how many spaces we need to use to clear the line (21 for the date and time prefix)
			spacesToClear = len(sprint) + 21
		}

	}
}

func getStatus(s State) string {

	return fmt.Sprintf("Tested: %d Speed(2s): %d/s Speed: %d/s",
		atomic.LoadUint64(s.TotalTested),
		atomic.LoadUint64(s.PerSecondShort),
		atomic.LoadUint64(s.PerSecondLong),
	)
}

func StatsTracker(state State) {
	tick := time.NewTicker(time.Second * 2)
	testedBefore := atomic.LoadUint64(state.TotalTested)
	timeBefore := time.Now()
	for _ = range tick.C {
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
