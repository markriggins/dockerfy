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

func runCmd(ctx context.Context, cancel context.CancelFunc, cmd *exec.Cmd, cancel_when_finished bool) {
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
    if debugFlag && cmd.SysProcAttr != nil && cmd.SysProcAttr.Credential != nil {
        log.Printf("command running as uid %d", cmd.SysProcAttr.Credential.Uid)
    }

	// Setup signaling -- a separate channel for goroutine for each command
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)

	wg.Add(1)
	go func() {
		defer wg.Done()
        for {
            select {
            case sig, ok := <-sigs:
                if !ok {
                    return
                }
                if debugFlag {
                    if sig != nil {
                        log.Printf("Command `%s` received signal %s", toString(cmd), sig)
                    } else {
                        log.Printf("Command `%s` done waiting for signals", toString(cmd))
                        return
                    }
                }
                if sig == nil {
                    signalProcessWithTimeout(cmd, syscall.SIGTERM)
                    signalProcessWithTimeout(cmd, syscall.SIGKILL)
                    return
                } else {
                    // Pass signals thru to children, let them decide how to handle it.
                    signalProcessWithTimeout(cmd, sig)
                }
            case <-ctx.Done():
                if debugFlag {
                    log.Printf("Command `%s` done waiting for signals (ctx.Done())", toString(cmd))
                }
                signalProcessWithTimeout(cmd, syscall.SIGTERM)
                signalProcessWithTimeout(cmd, syscall.SIGKILL)
                return
            }
        }
	}()

	err = cmd.Wait()
    signal.Stop(sigs)
    close(sigs)

	if err == nil {
		if verboseFlag {
			log.Printf("Command finished successfully: `%s`\n", toString(cmd))
		}
		if cancel_when_finished {
			cancel()
		}
	} else {
		log.Printf("Command `%s` exited with error: %s\n", toString(cmd), err)
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if exitCode == 0 {
					exitCode = status.ExitStatus()
				}
			}
			if exitCode == 0 {
				// If platform-specific exit_code cannot be determined exit with
				// with generic 1 for failure
				exitCode = 1
			}
		}
		cancel()
		// OPTIMIZE: This could be cleaner
		// os.Exit(err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus())
	}
}

func signalProcessWithTimeout(cmd *exec.Cmd, sig os.Signal) {
	done := make(chan struct{})

	go func() {
		cmd.Process.Signal(sig)
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
