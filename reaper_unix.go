// +build !windows,!solaris

package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
)

//
// Receive all SIGCHLDs, just to clear the signals
// NOTE: if we are "init" (with os.Getpid() == 1 ), then
//       we will ultimately inherit all orphaned grandchildren.
//

func _receiveSignals(ctx context.Context) {
	defer wg.Done()
	var signalsCh = make(chan os.Signal, 3)
	signal.Notify(signalsCh, unix.SIGCHLD)

	for {
		select {
		case <-ctx.Done():
			return
		case <-signalsCh:
			log.Println("Reaper: received a SIGCHLD")
		}
	}
}

//
// Reap all child processes by receiving their signals and
// waiting for their exit status
//
func ReapChildren(ctx context.Context, pollInterval time.Duration) {
	defer wg.Done()

	if os.Getpid() == 1 {
		log.Println("Reaper: init process reaper started")
	} else {
		log.Println("Reaper: started")
	}

	wg.Add(1)
	go _receiveSignals(ctx)

	if verboseFlag {
		log.Printf("Reaper: Polling every: %v", pollInterval)
	}

	// Loop forever, reaping zombies
	for {
		// Blocking on Wait4 may affect other child processes that are also trying to exec
		// and wait for their own children.  So we do this sparsely and intermittantly instead.
		select {
		case <-ctx.Done():
			log.Printf("Reaper: Done")
			return
		case <-time.After(pollInterval):
		}
		log.Printf("Reaper: looking for zombies")

		// Reap all zombie's that we have inherited (at this time)
		var zombiesReaped = 0
		{
		NonBlockingReaperLoop: // No blocking operations allowed in this loop
			for {
				var status unix.WaitStatus
				pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
				switch err {
				case nil:
					if pid > 0 && verboseFlag {
						zombiesReaped++
						log.Printf("Reaper: Reaped pid %d\n", pid)
						// Killed one Zombie -- look for another
					} else {
						// Try again later
						break NonBlockingReaperLoop
					}
				case unix.ECHILD:
					if verboseFlag {
						log.Println("Reaper: No more children at this time")
					}
					// No more zombies at this time, try again later
					break NonBlockingReaperLoop
				case unix.EINTR:
					if verboseFlag {
						log.Println("Reaper: Interrupted")
					}
					// Unlikely with WNOHANG, but possible, try again immediately
				default:
					if verboseFlag {
						log.Println("Reaper: Unexpected error", err)
					}
					// Unexpected err, log it and try again later
					break NonBlockingReaperLoop
				}
			}
		}
		log.Printf("Reaper: Reaped %d zombies", zombiesReaped)
	}
}
