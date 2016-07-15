package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
    "regexp"
    "strconv"

    //"github.com/kr/pretty"
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

	for i, arg := range cmd.Args {
		cmd.Args[i] = string_template_eval(arg)
	}

	// start the cmd
	err := cmd.Start()
	if err != nil {
		// TODO: bubble the platform-specific exit code of the process up via global exitCode
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

	cmd_err := cmd.Wait()

	if cmd_err == nil {
		if verboseFlag {
			log.Printf("Command finished successfully: `%s`\n", toString(cmd))
		}
	} else {
        re := regexp.MustCompile(`([0-9]+)`) // want to know what is in front of 'at'
        res := re.FindAllString(cmd_err.Error(), 1)
        exit_code := 1
        if len(res) > 0 {
            exit_code, _ = strconv.Atoi(res[0])
            exitCode = exit_code
            log.Printf("Command `%s` exited with exit_code: %d\n", toString(cmd), exit_code)
        } else {
            log.Printf("Command `%s` exited with error: %s\n", toString(cmd), cmd_err)
        }
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
