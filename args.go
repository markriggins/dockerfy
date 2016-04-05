package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

//
// Removes start and run commands and their arguments from os.Args
// Returns array of removed commands
//
func removeCmdFromOsArgs(flag string) []*exec.Cmd {

	var newOsArgs = []string{}
	var cmd *exec.Cmd
	var cmds = []*exec.Cmd{}

	for i := 0; i < len(os.Args); i++ {

		switch {
		case "-"+flag == os.Args[i] && cmd == nil:
			cmd = &exec.Cmd{Stdout: os.Stdout, Stderr: os.Stderr}
			cmds = append(cmds, cmd)
		case "--" == os.Args[i] && cmd != nil: // End of args for this cmd
			cmd = nil
		default:
			if cmd != nil {
				if len(cmd.Path) == 0 {
					cmd.Path = os.Args[i]
				}
				cmd.Args = append(cmd.Args, os.Args[i])
			} else {
				newOsArgs = append(newOsArgs, os.Args[i])
			}
		}
	}
	if cmd != nil {
		if len(cmd.Path) == 0 {
			log.Fatalf("need a command after -" + flag)
		}
		cmds = append(cmds, cmd)
	}
	os.Args = newOsArgs
	return cmds
}

func toString(cmd *exec.Cmd) string {
	s := ""
	for _, arg := range cmd.Args {
		s += arg + " "
	}
	return strings.TrimSpace(s)
}
