package librecursebuster

import (
	"testing"
)

func TestManageRequests(t *testing.T) { /*
		cfg := &Config{}
		state := &State{}
		wg := &sync.WaitGroup{}
		pages := make(chan SpiderPage, 5)
		newPages := make(chan SpiderPage, 5)
		confirmed := make(chan SpiderPage, 5)
		workers := make(chan struct{}, 1)
		printChan := make(chan OutLine, 1)
		maxDirs := make(chan struct{}, 1)
		testChan := make(chan string)

		go ManageRequests(cfg, state, wg, pages, newPages, confirmed, workers, printChan, maxDirs, testChan)
	*/
	//reads from page
	//checks if in blacklist
	//blacklist tests
	//check if url in blacklist is visited
	//for each method
	//if nobase is not set (does not send a request to the specified url - eg.)
	//checks if the page ends with a /, and if a wordlist has been specified
	//sends request to dirbing
}

func TestManageNewURLs(t *testing.T) {
	//todo: write test
}

func TestTestURL(t *testing.T) {
	//todo: write test
}

func TestDirBust(t *testing.T) {
	//todo:write test
}
