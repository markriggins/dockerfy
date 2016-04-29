package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/context"
)

func runCmd(ctx context.Context, cancel context.CancelFunc, cmd *exec.Cmd) {
	defer wg.Done()

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := copySecrets(cmd); err != nil {
		log.Fatalf("Could not copy secrets file", err)
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

// http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang/21067803#21067803
// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}

// Note that secrets file is typically readable only the root user
//
// If this command is running under a different user account, then copy the SECRETS_FILE
// and --secret files into the home directory .secrets (as an emphermeral file)
// and make them readable by the by the user account in case the application wants
// to read the file directly, instead of just using a template to alter a config file.
//
// Set the cmd.Env['SECRETS_FILE'] to point to the new, readable copy
func copySecrets(cmd *exec.Cmd) error {

	// If there is no Credential.Uid or its root, then there is no need to copy the secrets files
	// because the root user can always read them.
	if cmd.SysProcAttr == nil || cmd.SysProcAttr.Credential == nil || cmd.SysProcAttr.Credential.Uid == 0 {
		return nil
	}

	// If we are not running in a container, then do not copy secrets files because the copies
	// will not be ephemeral
	if _, err := os.Stat("/.dockerenv"); os.IsNotExist(err) {
		return nil
	}

	if cmdUser, err := user.LookupId(strconv.Itoa(int(cmd.SysProcAttr.Credential.Uid))); err != nil {
		return err
	} else {
		if os.Getenv("SECRETS_FILE") != "" {
			cmd.Env = make([]string, len(os.Environ())+1, len(os.Environ())+1)
			for i, envLine := range os.Environ() {
				if strings.HasPrefix(envLine, "SECRETS_FILE=") {
					cmd.Env[i] = "SECRETS_FILE=" + cmdUser.HomeDir +
						"/.secrets/" + filepath.Base(os.Getenv("SECRETS_FILE"))
				} else {
					cmd.Env[i] = envLine
				}
			}
		}
		secretsDir := cmdUser.HomeDir + "/.secrets"
		if _, err := os.Stat(secretsDir); os.IsNotExist(err) {
			// path/to/whatever does not exist
			if err := os.Mkdir(secretsDir, 0700); err != nil {
				return err
			}
			if err := os.Chown(secretsDir, int(cmd.SysProcAttr.Credential.Uid), int(cmd.SysProcAttr.Credential.Gid)); err != nil {
				return err
			}

			secretsFileNames := secretsFlag[:]
			if os.Getenv("SECRETS_FILE") != "" {
				secretsFileNames = append(secretsFileNames, os.Getenv("SECRETS_FILE"))
			}
			for _, secretsFileName := range secretsFileNames {
				copyName := secretsDir + "/" + filepath.Base(secretsFileName)
				if err := copyFileContents(secretsFileName, copyName); err != nil {
					return err
				}
				if err := os.Chown(copyName, int(cmd.SysProcAttr.Credential.Uid), int(cmd.SysProcAttr.Credential.Gid)); err != nil {
					return err
				}
				if err := os.Chmod(copyName, 0400); err != nil {
					return err
				}
			}
		}

	}

	return nil
}
