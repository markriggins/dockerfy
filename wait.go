package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

//
// wait for dependencies enumerated by the -wait option
//
func waitForDependencies() {
	dependencyChan := make(chan struct{})

	if waitFlag == nil {
		return
	}

	go func() {
		for _, host := range waitFlag {
			log.Println("Waiting for host:", host)
			u, err := url.Parse(host)
			if err != nil {
				log.Fatalf("bad hostname provided: %s. %s", host, err.Error())
			}

			switch u.Scheme {
			case "tcp", "tcp4", "tcp6":
				wg.Add(1)
				go func() {
					defer wg.Done()
					for {
						conn, _ := net.DialTimeout(u.Scheme, u.Host, waitTimeoutFlag)
						if conn != nil {
							log.Println("Connected to", u.String())
							return
						}
					}
				}()
			case "http", "https":
				wg.Add(1)
				go func() {
					defer wg.Done()
					for {
						resp, err := http.Get(u.String())
						if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
							log.Printf("Received %d from %s\n", resp.StatusCode, u.String())
							return
						}
					}
				}()
			default:
				log.Fatalf("invalid host protocol provided: %s. supported protocols are: tcp, tcp4, tcp6, http and https", u.Scheme)
			}
		}
		wg.Wait()
		close(dependencyChan)
	}()

	select {
	case <-dependencyChan:
		break
	case <-time.After(waitTimeoutFlag):
		log.Fatalf("Timeout after %s waiting on dependencies to become available: %v", waitTimeoutFlag, waitFlag)
	}

}
