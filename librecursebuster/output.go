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
			PrintOutput(printS, Good, 0, wg, printChan)
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
	maxLen := 0
	testedURL := ""
	for {
		spaces := ""
		select {
		case o := <-printChan:
			if o.Type != Status {
				if maxLen < len(o.Content) {
					maxLen = len(o.Content)
				}

				spaceCount := maxLen - len(o.Content)

				if spaceCount > 0 {
					if !cfg.NoStatus {
						spaces = strings.Repeat(" ", spaceCount)
					}
				}
				if o.Type == Debug {
					if cfg.VerboseLevel >= o.Level {
						o.Type.Println(o.Content + spaces)
					}
				} else {
					o.Type.Println(o.Content + spaces)
				}

			}
			wg.Done()
		case <-tick.C:
			status = getStatus(state)
		case t := <-testChan:
			status = getStatus(state)
			testedURL = t

		}

		sprint := fmt.Sprintf("%s %s", status, testedURL)
		if maxLen < len(sprint) {
			maxLen = len(sprint)
		}
		spaceCount := maxLen - len(sprint)
		if spaceCount > 0 {
			spaces = strings.Repeat(" ", spaceCount)
		}
		if !cfg.NoStatus {
			Status.Printf(sprint + spaces + "\r")
		}

	}
}

func getStatus(s State) string {

	return fmt.Sprintf("Total Tested: %d Short Speed: %d/s Long Speed: %d/s",
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
