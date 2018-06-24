package librecursebuster

import (
	"bufio"
	"os"
)

func LoadWords(filePath string, destChan chan string, printChan chan OutLine) {
	defer close(destChan)
	//get words from local file
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		destChan <- scanner.Text()
	}

}
