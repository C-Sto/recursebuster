package librecursebuster

import (
	"bufio"
	"fmt"
	"os"
)

func LoadWords(filePath string, destChan chan string, printChan chan OutLine) {
	defer close(destChan)
	//get words from local file
	file, err := os.Open(filePath)
	if err != nil {
		printChan <- OutLine{
			Content: fmt.Sprintf("Error Loading Words: %s", err),
			Type:    Error,
		}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		destChan <- scanner.Text()
	}

}
