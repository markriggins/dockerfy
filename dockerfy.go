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
)

// Flags
var (
	delimsFlag           string
	overlaysFlag         sliceVar
	logPollFlag          bool
	reapPollIntervalFlag time.Duration
	reapFlag             bool
	runsFlag             sliceVar
	secretsFlag          string
	startsFlag           sliceVar
	stderrTailFlag       sliceVar
	stdoutTailFlag       sliceVar
	templatesFlag        sliceVar
	verboseFlag          bool
	versionFlag          bool
	waitFlag             hostFlagsVar
	waitTimeoutFlag      time.Duration
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
	println(`Usage: dockerfy [options] [command]

Utility to simplify running applications in docker containers

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
       dockerfy -template nginx.tmpl:/etc/nginx/nginx.conf \
   	     -overlay overlays/_common/html:/usr/share/nginx/ \
   	     -overlay overlays/$DEPLOYMENT_ENV/html:/usr/share/nginx/ \`)
	println(`   	     -stdout /var/log/nginx/access.log \
             -stderr /var/log/nginx/error.log \
             -wait tcp://web:8000 nginx \
             -secrets /secrets/secrets.env
	`)
	println(`   Run a command and reap any zombie children that the command forgets to reap

       dockerfy -reap command 
	     `)
	println(`   Run /bin/echo before the main command runs:
       
       dockerfy -run /bin/echo -e "Starting -- command\n\n" 
	     `)

	println(`   Start /bin/service before the main command runs and exit if the service fails:
       
       dockerfy -start /bin/sleep 5 -- /bin/service 
	     `)
	println(`For more information, see https://github.com/markriggins/dockerfy`)
}

func main() {

	log.SetPrefix("dockerfy: ")

	flag.BoolVar(&versionFlag, "version", false, "show version")
	flag.BoolVar(&logPollFlag, "log-poll", false, "use polling to tail log files")
	flag.Var(&templatesFlag, "template", "Template (/template:/dest). Can be passed multiple times")
	flag.Var(&overlaysFlag, "overlay", "overlay (/src:/dest). Can be passed multiple times")
	flag.StringVar(&secretsFlag, "secrets", "", "secrets (path to secrets.env file)")
	flag.Var(&runsFlag, "run", "run (cmd [opts] [args] --) Can be passed multiple times")
	flag.Var(&startsFlag, "start", "start (cmd [opts] [args] --) Can be passed multiple times")
	flag.BoolVar(&reapFlag, "reap", false, "reap all child processes")
	flag.BoolVar(&verboseFlag, "verbose", false, "verbose output")
	flag.Var(&stdoutTailFlag, "stdout", "Tails a file to stdout. Can be passed multiple times")
	flag.Var(&stderrTailFlag, "stderr", "Tails a file to stderr. Can be passed multiple times")
	flag.StringVar(&delimsFlag, "delims", "", `template tag delimiters. default "{{":"}}" `)
	flag.Var(&waitFlag, "wait", "Host (tcp/tcp4/tcp6/http/https) to wait for before this container starts. Can be passed multiple times. e.g. tcp://db:5432")
	flag.DurationVar(&waitTimeoutFlag, "timeout", 10*time.Second, "Host wait timeout")
	flag.DurationVar(&reapPollIntervalFlag, "reap-poll-interval", 120*time.Second, "Polling interval for reaping zombies")

	var startCmds = removeCmdFromOsArgs("start")
	var runCmds = removeCmdFromOsArgs("run")

	flag.Usage = usage
	flag.Parse()

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
		if strings.Contains(o, ":") {
			parts := strings.Split(o, ":")
			if len(parts) != 2 {
				log.Fatalf("bad overlay argument: '%s'. expected \"/src:/dest\"", o)
			}
			src, dest := os.ExpandEnv(string_template_eval(parts[0])), os.ExpandEnv(string_template_eval(parts[1]))
			if _, err := os.Stat(src); os.IsNotExist(err) {
				log.Printf("overlay source: %s does not exist.  Skipping", src)
				continue
			}
			log.Printf("overlaying %s --> %s", src, dest)

			var cmd *exec.Cmd

			if strings.HasSuffix(src, "/") {
				src += "*"
			}
			if matches, err := filepath.Glob(src); err == nil {
				for _, dir := range matches {
					cmd = exec.Command("cp", "-rv", dir, dest)
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
			template, dest = os.ExpandEnv(string_template_eval(parts[0])), os.ExpandEnv(string_template_eval(parts[1]))
		}
		generateFile(template, dest)
	}

	waitForDependencies()

	// Setup context
	ctx, cancel = context.WithCancel(context.Background())

	for _, logFile := range stdoutTailFlag {
		wg.Add(1)
		go tailFile(ctx, cancel, logFile, logPollFlag, os.Stdout)
	}

	for _, logFile := range stderrTailFlag {
		wg.Add(1)
		go tailFile(ctx, cancel, logFile, logPollFlag, os.Stderr)
	}

	// Process -start and -run flags
	for _, cmd := range runCmds {

		log.Printf("Pre-Running: `%s`\n", toString(cmd))
		wg.Add(1)
		// Run in the foreground
		runCmd(ctx, cancel, cmd.Path, cmd.Args[1:]...)
	}
	for _, cmd := range startCmds {
		log.Printf("Starting Service: `%s`\n", toString(cmd))
		wg.Add(1)
		go runCmd(ctx, func() {
			log.Printf("Service `%s` stopped\n", toString(cmd))
			cancel()
		}, cmd.Path, cmd.Args[1:]...)
	}

	if flag.NArg() > 0 {
		var cmdString = strings.Join(flag.Args(), " ")
		log.Printf("Running Primary Command: `%s`\n", cmdString)
		wg.Add(1)
		go runCmd(ctx, func() {
			log.Printf("Primary Command `%s` stopped\n", cmdString)
			cancel()
		}, flag.Arg(0), flag.Args()[1:]...)
	}

	if reapFlag {
		wg.Add(1)
		go ReapChildren(ctx, reapPollIntervalFlag)
	}

	wg.Wait()

}
