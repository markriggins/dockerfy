package main

import (
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type Commands struct {
	run        []*exec.Cmd         // list of commands to run BEFORE the primar
	start      []*exec.Cmd         // list of services to start
	credential *syscall.Credential // credentials for primary command
}

//
// Removes --start and --run commands options and arguments from os.Args
// Removes --user <uid|username> options and applies the credentials to following
//           start or run commands and primary command
// Returns array of removed run commands, and an array of removed start commands
//
func removeCommandsFromOsArgs() Commands {

	var newOsArgs = []string{}
	var commands = Commands{}

	var cmd *exec.Cmd
	var cmd_user *user.User

	for i := 0; i < len(os.Args); i++ {
		switch {
		case ("--start" == os.Args[i] || "-start" == os.Args[i]) && cmd == nil:
			cmd = &exec.Cmd{Stdout: os.Stdout,
				Stderr:      os.Stderr,
				SysProcAttr: &syscall.SysProcAttr{Credential: commands.credential}}
			commands.start = append(commands.start, cmd)

		case ("--run" == os.Args[i] || "-run" == os.Args[i]) && cmd == nil:
			cmd = &exec.Cmd{Stdout: os.Stdout,
				Stderr:      os.Stderr,
				SysProcAttr: &syscall.SysProcAttr{Credential: commands.credential}}
			commands.run = append(commands.run, cmd)

		case ("--user" == os.Args[i] || "-user" == os.Args[i]) && cmd == nil:
			if os.Getuid() != 0 {
				log.Fatalf("dockerfy must run as root to use the --user flag")
			}
			cmd_user = &user.User{}

		case "--" == os.Args[i] && cmd != nil: // End of args for this cmd
			cmd = nil

		default:
			if cmd_user != nil {
				// Expect a username or uid
				var err1 error
				cmd_user, err1 = user.LookupId(os.Args[i])
				if cmd_user == nil {
					// Not a userid, try as a username
					cmd_user, err1 = user.Lookup(os.Args[i])
					if cmd_user == nil {
						log.Fatalf("unknown user: '%s': %s", os.Args[i], err1)
					}
				}
				uid, _ := strconv.Atoi(cmd_user.Uid)
				gid, _ := strconv.Atoi(cmd_user.Gid)

				commands.credential = new(syscall.Credential)
				commands.credential.Uid = uint32(uid)
				commands.credential.Gid = uint32(gid)

				cmd_user = nil
			} else if cmd != nil {
				// Expect a command first, then a series of arguments
				if len(cmd.Path) == 0 {
					_ = "breakpoint"
					cmd.Path = os.Args[i]
					if filepath.Base(cmd.Path) == cmd.Path {
						cmd.Path, _ = exec.LookPath(cmd.Path)
					}
				}
				cmd.Args = append(cmd.Args, os.Args[i])
			} else {
				newOsArgs = append(newOsArgs, os.Args[i])
			}
		}
	}
	if cmd_user != nil {
		log.Fatalln("need a username or uid after the --user flag")
	}
	if cmd != nil {
		log.Fatalf("need a command after the --start or --run flag")
	}
	os.Args = newOsArgs
	return commands
}

func toString(cmd *exec.Cmd) string {
	s := ""
	for _, arg := range cmd.Args {
		s += arg + " "
	}
	return strings.TrimSpace(s)
}
