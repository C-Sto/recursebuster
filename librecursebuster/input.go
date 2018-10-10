package librecursebuster

import (
	"bufio"
	"os"
)

//LoadWords asynchronously loads in words to a channel.
//Expects the channel to either be big enough to load the whole file, or that it will be streamed from as the file is opened and read from.
func LoadWords(filePath string, destChan chan string) {
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
