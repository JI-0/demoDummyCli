package main

import (
	"bufio"
	"log"
	"os"
	"sync"
)

var timestampF *os.File
var timestampW *bufio.Writer
var chanel chan []byte

var setupOnce sync.Once

func timestamp() chan []byte {
	setupOnce.Do(func() {
		// Set up a timestamp file
		new_file, new_file_err := os.OpenFile("timestamps.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if new_file_err != nil {
			log.Fatal(new_file_err)
		}
		// defer new_file.Close()
		timestampW = bufio.NewWriter(new_file)
		// defer writer.Flush()
		timestampF = new_file
		chanel = make(chan []byte)
		go timestampWrite()
	})
	return chanel
}

func timestampWrite() {
	for {
		select {
		case entry, ok := <-chanel:
			if !ok {
				log.Println("TIMESTAMP ERROR!!!!")
				timestampF.Close()
				timestampW.Flush()
				return
			}
			timestampW.WriteString(string(entry))
		}
	}
}
