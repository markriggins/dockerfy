package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/context"
)

func runCmd(ctx context.Context, cancel context.CancelFunc, cmd *exec.Cmd) {
	defer wg.Done()

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := copySecretsFiles(cmd); err != nil {
		log.Fatalf("Could not copy secrets files", err)
	}

	// start the cmd
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Error starting command: `%s` - %s\n", toString(cmd), err)
	}

	// Setup signaling -- a separate channel for goroutine for each command
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case sig := <-sigs:
			if verboseFlag {
				log.Printf("Received signal: %s\n", sig)
			}
			signalProcessWithTimeout(cmd, sig)
			if cancel != nil {
				cancel()
			}
		case <-ctx.Done():
			if verboseFlag {
				log.Printf("Done waiting for signals")
			}
			// exit when context is done
		}
	}()

	err = cmd.Wait()

	if err == nil {
		if verboseFlag {
			log.Printf("Command finished successfully: `%s`\n", toString(cmd))
		}
	} else {
		log.Printf("Command `%s` exited with error: %s\n", toString(cmd), err)
		// OPTIMIZE: This could be cleaner
		// os.Exit(err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus())
	}
	if cancel != nil {
		cancel()
	}
}

func signalProcessWithTimeout(cmd *exec.Cmd, sig os.Signal) {
	done := make(chan struct{})

	go func() {
		cmd.Process.Signal(sig) // pretty sure this doesn't do anything. It seems like the signal is automatically sent to the command?
		cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(10 * time.Second):
		log.Println("Killing command due to timeout.")
		cmd.Process.Kill()
	}
}
