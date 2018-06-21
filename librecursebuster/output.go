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
	fmt.Println("GoRecurseBuster V" + cfg.Version)
	fmt.Println("Poorly hacked together by C_Sto")
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
			printChan <- OutLine{
				Content: printS,
				Type:    Good,
			}
		}
		wg.Done()
	}
}

func StatusPrinter(testedTotal *uint64, printChan chan OutLine) {
	tick := time.NewTicker(time.Second * 2)
	status := getStatus(testedTotal)
	for {
		select {
		case o := <-printChan:
			if o.Type != Status {
				o.Type.Println(o.Content)
			}
		case <-tick.C:
			status = getStatus(testedTotal)
		}
		Status.Printf(status)
	}
}

var lastTick = time.Now()
var lastTested = float64(0)

func getStatus(testedTotal *uint64) string {
	tested := atomic.LoadUint64(testedTotal)
	thisTick := time.Now()
	seconds := time.Since(lastTick).Seconds()
	testedInDuration := float64(tested) - lastTested
	persecond := testedInDuration / seconds
	lastTick = thisTick
	lastTested = float64(tested)
	return fmt.Sprintf("Total Tested: %d Estimated speed: %d/s\r", tested, int64(persecond)) //this is extremely rough and probably wrong
}
