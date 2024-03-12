package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

func getOutFile(filename string) (*os.File, func() error, error) {
	outFile, err := os.Create(filename)
	if err != nil {
		return nil, nil, err
	}
	return outFile, func() error {
		return outFile.Close()
	}, nil
}

func runResultWorker(results <-chan string, outFile *os.File, done chan<- bool) {
	for res := range results {
		_, err := io.WriteString(outFile, fmt.Sprintf("%s\n", res))
		if err != nil {
			log.Println(err)
		}
	}
	done <- true
}
