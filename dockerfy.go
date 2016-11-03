// Dockerfy -- utility program to simplify running applications in docker containers
//
// Typical Usage (inside a Dockerfile)
//   ENTRYPOINT [ "dockerfy", \
//     "-template", "/etc/nginx/conf.d/default.conf.tmpl:/etc/nginx/conf.d/default.conf", \
//     "-stdout", "/var/log/nginx/access.log", \
//     "-stderr", "/var/log/nginx/error.log", \
//     "-secrets-files", "/secrets/secrets.env", \
//     "--" ],
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/context"
)

type sliceVar []string
type hostFlagsVar []string

var (
	buildVersion string
	cancel       context.CancelFunc
	ctx          context.Context
	delims       []string
	wg           sync.WaitGroup
	exitCode     int
)

// Flags
var (
	delimsFlag           string
	overlaysFlag         sliceVar
	logPollFlag          bool
	reapPollIntervalFlag time.Duration
	reapFlag             bool
	runsFlag             sliceVar
	secretsFilesFlag     sliceVar
	startsFlag           sliceVar
	stderrTailFlag       sliceVar
	stdoutTailFlag       sliceVar
	templatesFlag        sliceVar
	usersFlag            sliceVar
    verboseFlag          bool
    debugFlag            bool
	versionFlag          bool
	waitFlag             hostFlagsVar
	waitTimeoutFlag      time.Duration
	helpFlag             bool
)

func (i *hostFlagsVar) String() string {
	return fmt.Sprint(*i)
}

func (i *hostFlagsVar) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (s *sliceVar) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *sliceVar) String() string {
	return strings.Join(*s, ",")
}

func usage() {
	println(`Usage: dockerfy [options] command [command options]

Try --help for a list of options and helpful examples
		`)
}

func help() {

	println("Dockerfy -- a utility to initialize and run applications in docker containers ")
	usage()
	println(`

Options:`)
	flag.PrintDefaults()

	println(`
Arguments:
  command - command to be executed
  `)

	println(`Examples:
`)
	println(`   Generate /etc/nginx/nginx.conf using nginx.tmpl as a template, tail /var/log/nginx/access.log
   and /var/log/nginx/error.log, waiting for a website to become available on port 8000 and start nginx:`)
	println(`
       dockerfy --template nginx.tmpl:/etc/nginx/nginx.conf \
   	     --overlay overlays/_common/html:/usr/share/nginx/ \
   	     --overlay overlays/$DEPLOYMENT_ENV/html:/usr/share/nginx/ \`)
	println(`   	     --stdout /var/log/nginx/access.log \
             --stderr /var/log/nginx/error.log \
             --wait tcp://web:8000 nginx \
             --secrets-files /secrets/secrets.env
	`)
	println(`   Run a command and reap any zombie children that the command forgets to reap

       dockerfy --reap command
	     `)
	println(`   Run /bin/echo before the main command runs:

       dockerfy --run /bin/echo -e "Starting -- command\n\n"
	     `)

	println(`   Run or start all subsequent commands as user 'nginx'

       dockerfy --user nginx /usr/bin/id
	     `)

	println(`   Start /bin/service before the main command runs and exit if the service fails:

       dockerfy --start /bin/sleep 5 -- /bin/service
	     `)
	println(`For more information, see https://github.com/markriggins/dockerfy `)
}

func main() {

	exitCode = 0
	log.SetPrefix("dockerfy: ")

	// Bug on OS X beta Docker version 1.12.0-rc3, build 91e29e8, experimental
	// cannot resolve link names that do not appear in /etc/hosts w/o using cgo.
	// Setting this env var forces the use of cgo
	//if os.Getenv("GODEBUG") != "" {
	//    os.Setenv("GODEBUG", "netdns=cgo")
	//}

	flag.BoolVar(&versionFlag, "version", false, "show version")
	flag.BoolVar(&helpFlag, "help", false, "print help message")
	flag.BoolVar(&logPollFlag, "log-poll", false, "use polling to tail log files")
	flag.Var(&templatesFlag, "template", "Template (/template:/dest). Can be passed multiple times")
	flag.Var(&overlaysFlag, "overlay", "overlay (/src:/dest). Can be passed multiple times")
	flag.Var(&secretsFilesFlag, "secrets-files", "secrets files (path to secrets.env files). Colon-separated list")
	flag.Var(&runsFlag, "run", "run (cmd [opts] [args] --) Can be passed multiple times")
	flag.Var(&startsFlag, "start", "start (cmd [opts] [args] --) Can be passed multiple times")
	flag.BoolVar(&reapFlag, "reap", false, "reap all zombie processes")
    flag.BoolVar(&verboseFlag, "verbose", false, "verbose output")
    flag.BoolVar(&debugFlag, "debug", false, "debugging output")
	flag.Var(&stdoutTailFlag, "stdout", "Tails a file to stdout. Can be passed multiple times")
	flag.Var(&stderrTailFlag, "stderr", "Tails a file to stderr. Can be passed multiple times")
	flag.StringVar(&delimsFlag, "delims", "", `template tag delimiters. default "{{":"}}" `)
	flag.Var(&waitFlag, "wait", "Host (tcp/tcp4/tcp6/http/https) to wait for before this container starts. Can be passed multiple times. e.g. tcp://db:5432")
	flag.DurationVar(&waitTimeoutFlag, "timeout", 10*time.Second, "Host wait timeout duration, defaults to 10s")
	flag.DurationVar(&reapPollIntervalFlag, "reap-poll-interval", 120*time.Second, "Polling interval for reaping zombies")

    // Manually pre-process the --debug and --verbose flags so we can debug our complex argument pre-processing
    // that happens BEFORE flag.Parse()
    for i := 0; i < len(os.Args); i++ {
        if strings.TrimSpace(os.Args[i]) == "--debug" {
            debugFlag = true
            log.Printf("debugging output ..")
        } else if strings.TrimSpace(os.Args[i]) == "--verbose" {
            verboseFlag = true
        }
    }

	var commands = removeCommandsFromOsArgs()

	flag.Usage = usage

	flag.Parse()

	if helpFlag {
		help()
		os.Exit(1)
	}

	if versionFlag {
		fmt.Println(buildVersion)
		return
	}

	if flag.NArg() == 0 && flag.NFlag() == 0 {
		usage()
		os.Exit(1)
	}

	if delimsFlag != "" {
		delims = strings.Split(delimsFlag, ":")
		if len(delims) != 2 {
			log.Fatalf("bad delimiters argument: %s. expected \"left:right\"", delimsFlag)
		}
	}

	// Overlay files from src --> dst
	for _, o := range overlaysFlag {
        if debugFlag {
            log.Printf("--overlay: %s", o)
        }
		if strings.Contains(o, ":") {
			parts := strings.Split(o, ":")
			if len(parts) != 2 {
				log.Fatalf("bad overlay argument: '%s'. expected \"/src:/dest\"", o)
			}
			src, dest := string_template_eval(parts[0]), string_template_eval(parts[1])
			if _, err := os.Stat(src); os.IsNotExist(err) {
				log.Printf("overlay source: %s does not exist.  Skipping", src)
				continue
			}

			var cmd *exec.Cmd
			if strings.HasSuffix(src, "/") {
				src += "*"
			}

			if verboseFlag {
				log.Printf("overlaying %s --> %s", src, dest)
			}

			if matches, err := filepath.Glob(src); err == nil {
				for _, dir := range matches {
					cp_opts := "-r"
					if verboseFlag {
						cp_opts = "-rv"
					}
					cmd = exec.Command("cp", cp_opts, dir, dest)
					cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
					if err := cmd.Run(); err != nil {
						log.Fatal(err)
					}
				}
			} else {
				log.Printf("No matches for overlay source: '%s':%s", src, err)
			}
		}
	}

	for _, t := range templatesFlag {
		template, dest := t, ""
		if strings.Contains(t, ":") {
			parts := strings.Split(t, ":")
			if len(parts) != 2 {
				log.Fatalf("bad template argument: %s. expected \"/template:/dest\"", t)
			}
			template, dest = string_template_eval(parts[0]), string_template_eval(parts[1])
		}
		generateFile(template, dest)
	}

	waitForDependencies()

	// Setup context
	ctx, cancel = context.WithCancel(context.Background())

	// Process -run flags
	for _, cmd := range commands.run {

		if verboseFlag {
			log.Printf("Pre-Running: `%s`\n", toString(cmd))
		}
		// Run to completion, but do not cancel our ctx context unless we fail
		wg.Add(1)
		go runCmd(ctx, func() {
			log.Printf("--run command `%s` finished\n", toString(cmd))
			if exitCode != 0 {
				cancel()
			}
		}, cmd, false /*cancel_when_finished*/)
		wg.Wait()
        if exitCode != 0 {
            cancel()
            os.Exit(exitCode)
        }
	}

	for _, logFile := range stdoutTailFlag {
		wg.Add(1)
		go tailFile(ctx, cancel, string_template_eval(logFile), logPollFlag, os.Stdout)
	}

	for _, logFile := range stderrTailFlag {
		wg.Add(1)
		go tailFile(ctx, cancel, string_template_eval(logFile), logPollFlag, os.Stderr)
	}

	// Start the reaper
	if reapFlag {
		wg.Add(1)
		go ReapChildren(ctx, reapPollIntervalFlag)
	}

	// Process -start flags
	for _, cmd := range commands.start {
		if verboseFlag {
			log.Printf("Starting Service: `%s`\n", toString(cmd))
		}
		wg.Add(1)

		// Start each service, and bind them to our ctx context so
		// 1) any failure will close/cancel ctx
		// 2) if the primary command fails, then the services will be stopped
		go runCmd(ctx, func() {
			log.Printf("Service `%s` cancelled\n", toString(cmd))
			cancel()
		}, cmd, true /*cancel_when_finished*/)
	}

	if flag.NArg() > 0 {

		// perform template substitution on primary cmd
		//for i, arg := range flag.Args() {
		//	flag.Args()[i] = string_template_eval(arg)
		//}

		var cmdString = strings.Join(flag.Args(), " ")
		if verboseFlag {
			log.Printf("Running Primary Command: `%s`\n", cmdString)
		}
		wg.Add(1)

		primary_command := exec.Command(flag.Arg(0), flag.Args()[1:]...)
		primary_command.SysProcAttr = &syscall.SysProcAttr{Credential: commands.credential}
		go runCmd(ctx, func() {
			if verboseFlag {
				log.Printf("Primary Command `%s` finished\n", cmdString)
			}
			cancel()
		}, primary_command, true /*cancel_when_finished*/)

        //TODO -- catch signals and log the fact that dockerfy itself was terminated
	} else {
		cancel()
	}

	wg.Wait()

	os.Exit(exitCode)
}
