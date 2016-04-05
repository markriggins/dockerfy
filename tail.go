package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hpcloud/tail"
	"golang.org/x/net/context"
)

func tailFile(ctx context.Context, cancel context.CancelFunc, fileName string, poll bool, dest *os.File) {
	defer wg.Done()

	// Make sure we can read it -- apparently tail.TailFile does not complain if the file does not exist!!
	_file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Error opening '%s':%s\n", fileName, err)
		cancel()
	}
	_file.Close()

	t, err := tail.TailFile(fileName, tail.Config{
		Follow: true,
		ReOpen: true,
		Poll:   poll,
		Logger: tail.DiscardingLogger,
	})
	if err != nil {
		log.Printf("unable to tail %s: %s\n", fileName, err)
		cancel()
	}
	// main loop
	for {
		select {
		// if the channel is done, then exit the loop
		case <-ctx.Done():
			t.Stop()
			t.Cleanup()
			return
		// get the next log line and echo it out
		case line := <-t.Lines:
			if line != nil {
				fmt.Fprintln(dest, line.Text)
			}
		}
	}
}
