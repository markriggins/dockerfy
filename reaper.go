// +build windows,solaris

package main

import (
	"log"
	"time"

	"golang.org/x/net/context"
)

//
// Reap all child processes by receiving their signals and
// waiting for their exit status
//
func ReapChildren(ctx context.Context, pollInterval time.Duration) {
	defer wg.Done()
	log.Println("Reaper: Not supported by OS")
}
