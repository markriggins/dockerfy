![dockerfyme](https://github.com/markriggins/dockerfy/blob/master/dockerfyme.png)
dockerfy -- Utility to initialize docker containers
===================================================
**Dockerfy** is a utility program to initialize and control container applications, and also provide some
missing OS functionality (such as an init process, and reaping zombies etc.)

##Key Features

1. Overlays of alternative content at runtime
2. Templates for configuration and content
3. Environment Variable substitutions into templates and overlays
4. Secrets injected into configuration files (without leaking them to the environment)
5. Waiting for dependencies (any server and port) to become available before the primary command starts
6. Tailing log files to the container's stdout and/or stderr
7. Running commands before the primary command begins
8. Starting Services -- and shutting down the container if they fail
9. Propagating signals to child processes
10. Reaping Zombie (defunct) processes
11. Running services and commands under various user accounts


## Dockerfile Example

    FROM markriggins/nginx-with-dockerfy

    ENTRYPOINT [ "dockerfy",                                                                            \
                    "--secrets-files", "/secrets/secrets.env",                                          \
                    "--overlay", "/app/overlays/{{ .Env.DEPLOYMENT_ENV }}/html/:/usr/share/nginx/html",         \
                    "--template", "/app/nginx.conf.tmpl:/etc/nginx/nginx.conf",                         \
                    "--wait", 'tcp://{{ .Env.MYSQLSERVER }}:{{ .Env.MYSQLPORT }}', "--timeout", "60s",  \
                    "--run", '/app/bin/migrate_lock', "--server='{{ .Env.MYSQLSERVER }}:{{ .Env.MYSQLPORT }}'",  "--", \
                    "--start", "/app/bin/cache-cleaner-daemon", "-p", "{{ .Secret.DB_PASSWORD }}", "--",\
                    "--reap",                                                                           \
                    "--user", "nobody",                                                                 \
                  	"nginx",  "-g",  "daemon off;" ]

## equivalent docker-compose.yml Example

    nginx:
      image: markriggins/nginx-with-dockerfy

      volumes:
        - /secrets:/secrets

      environment:
        - SECRETS_FILES=/secrets/secrets.env

      entrypoint:
        - dockerfy

      command: [
        "--overlay", "/app/overlays/{{ .Env.DEPLOYMENT_ENV }}/html/:/usr/share/nginx/html",
        "--template", "/app/nginx.conf.tmpl:/etc/nginx/nginx.conf",
        "--wait", "tcp://{{ .Env.MYSQLSERVER }}:{{ .Env.MYSQLPORT }}", "--timeout", "60s",
        "--wait", "tcp://{{ .Env.MYSQLSERVER }}:{{ .Env.MYSQLPORT }}", "--timeout", "60s",
        "--run", "/app/bin/migrate_lock", "--server='{{ .Env.MYSQLSERVER }}:{{ .Env.MYSQLPORT }}'",  "--",
        "--start", "/app/bin/cache-cleaner-daemon", "-p", '{{ .Secret.DB_PASSWORD }}', "--",
        "--reap",
        "--user", "nobody",
        '--', 'nginx', '-g', 'daemon off;' ]



The above example will run the nginx program inside a docker container, but **before nginx starts**, **dockerfy** will:

1. **Sparsely Overlay** files from the application's /app/overlays directory tree from /app/overlays/${DEPLOYMENT_ENV}/html **onto** /usr/share/nginx/html.  For example, the robots.txt file might be restrictive in the "staging" deployment environment, but relaxed in "production", so the application can maintain separate copies of robots.txt for each deployment environment: /app/overlays/staging/robots.txt, and /app/overlays/production/robots.txt.   Overlays add or replace files similar to `cp -R` withou affecting other existing files in the target directory.
2. **Load secret settings** from a file a /secrets/secrets.env, that become available for use in templates as {{ .Secret.**VARNAME** }}
3. **Execute the nginx.conf.tmpl template**. This template uses the powerful go language templating features to substitute environment variables and secret settings directly into the nginx.conf file. (Which is handy since nginx doesn't read the environment itself.)  Every occurance of {{ .Env.**VARNAME** }} will be replaced with the value of $VARNAME, and every {{ .Secret.**VARNAME** }} will be replaced with the secret value of VARNAME.
4. **Wait** for the http://{{ .Env.MYSQLSERVER }} server to start accepting requests on port {{ .Env.MYSQLPORT }} for up to 60 seconds
5. **Run migrate_lock** a program to perform a Django/MySql database migration to update the database schema, and wait for it to finish. If **migrate_lock** fails, then dockerfy will exit with migrate_lock's exit code, and the primary command **nginx** will never start.
6. **Start the cache-cleaner-daemon**, which will run in the background presumably cleaning up stale cache files while nginx runs.  If for any reason the cache-cleaner-daemon exits, then dockerfy will also exit with the cache-cleaner-daemon's exit code.
7. **Start Reaping Zombie processes** under a separate goroutine in case the cache-cleaner-deamon loses track of its child processes.
8. **Run nginx** with its customized nginx.conf file and html as user `nobody`
9. **Propagate Signals** to all processes, so the container can exit cleanly on SIGHUP or SIGINT
10. **Monitor Processes** and exit if nginx or the cache-cleaner-daemon dies
11. **Exit** with the primary command's exit_code if the primary command finishes.


This all assumes that the /secrets volume was mounted and the environment variables $MYSQLSERVER, $MYSQLPORT
and $DEPLOYMENT_ENV were set when the container started.  Note that **dockerfy** expands the environment variables in its arguments, since the ENTRYPOINT [] form in Dockerfiles does not, replacing all {{ .Env.VARNAME }} and {{ .Secret.VARNAME }} occurances with their values from the environment or secrets files.

Note that the unexpanded argument '{{ .Secret.DB_PASSWORD }}', would be visible in `ps -ef` output, not the actual password

Note that ${VAR_NAME}'s are NOT expanded by dockerfy because docker-compose and ecs-cli also expand environment variables inside yaml files.  The {{ .Env.VAR_NAME }} form passes through easily, as long as it is inside a singly-quoted string

The "--" argument is used to signify the end of arguments for a --start or --run command.


# Typical Use-Case
The typical use case for **dockerfy** is when you have an
application that:

1. Relies strictly on configuration files to initialize itself. For example, ningx does not use environment variables directly inside nginx.conf files
2. Needs to wait for some other service to become available.  For example, in a docker-compose.yml application with a webserver and a database, the webserver may need to wait for the the database to initialize itself at start listening for logins before the webserver starts accepting requests, or tries to connect to the database.
3. Needs to run some initialization before the real application starts.  For example, applications that rely on a dedicated database may need to run a migrations script to update the database
4. Needs a companion service to run in the background, such as uwsgi, or a cleanup daeamon to purge caches.
5. Is a long-lived Container that runs a complex application.  For example, if the long-lived application forks a lot of child processes that forget to wait for their own children, then OS resources can consumed by defunct (zombie) processes, eventually filling the process table.
6. Needs Passwords or other Secrets.  For example, a Django server might need to login to a database, but passing the password through the environment or a run-time flag is susceptible to accidental leakage.

Another use case is when the application logs to specific files on the filesystem and not stdout
or stderr. This makes it difficult to troubleshoot the container using the `docker logs` command.
For example, nginx will log to `/var/log/nginx/access.log` and
`/var/log/nginx/error.log` by default. While you can work around this for nginx by replacing the access.log file with a symbolic link to /dev/stdout,  **dockerfy** offers a generic solution allowing you to specify which logs files should
be tailed and where they should be sent.

## Customizing Startup and Application Configuration

### Sparse Overlays
Overlays are used provide alternative versions of entire files for various deployment environments (or other reasons).  **[Unlike mounted volumes, overlays do not hide the existing directories and files, they copy the altenative content ONTO the existing content, replacing only what is necessary]**.  This comes in handy for *if-less* languages like CSS, robots.txt, and for icons and images that might need to change depending on the deployment environment.  To use overlays, the application can create a directory tree someplace, perhaps at ./overlays with subdirectories for the various deployment environents like this:

	overlays/
	├── _common
	│   └── html
	│       └── robots.txt
	├── prod
	│   └── html
	│       └── robots.txt
	└── staging
    	└── html
        	└── index.html

The entire ./overlays files must be COPY'd into the Docker image (usually along with the application itself):

	COPY / /app

Then the desired alternative for the files can be chosen at runtime use the --overlay *src:dest* option

	$ dockefy --overlay /app/overlays/_commmon/html:/usr/share/nginx/ \
		      --overlay /app/overlays/{{ .Env.DEPLOYMENT_ENV }}/html:/usr/share/nginx/ \
		    nginx

If the source path ends with a /, then all subdirectories underneath it will be copied.  This allows copying onto the root file system as the destination; so you can `-overlay /app/_root/:/` to copy files such as /app/_root/etc/nginx/nginx.conf --> /etc/nginx/nginx.conf.   This is handy if you need to drop a lot of files into various exact locations

Overlay sources that do not exist are simply skipped.  The allows you to specify potential sources of content that may or may not exist in the running container.  In the above example if $DEPLOYMENT_ENV environment variable is set to 'local' then the second overlaw will be skipped if there is no corresponding /app/overlays/local source directory, and the container will run with the '_common' html content.


#### Loading Secret Settings
Secrets can loaded from a file by using the `--secrets-files` option or the $SECRETS_FILES environment variable.   The secrets files ending with `.env` must contain simple NAME=VALUE lines, following bash shell conventions for definitions and comments. Leading and trailing quotes will be trimmed from the value.  Secrets files ending with `.json` will be loaded as JSON, and must be `a simple single-level dictionary of strings`

    #
    # These are our secrets
    #
    PROXY_PASSWORD="a2luZzppc25ha2Vk"

or secrets.json (which must be **a simple single-level dictionary of strings**)

    {
      "PROXY_PASSWORD": "a2luZzppc25ha2Vk"
    }

Secrets can be injected into configuration files by using [Secrets in Templates](https://github.com/markriggins/dockerfy#secrets-in-templates).

For convenience, all secrets files are combined into ~/.secrets/combined_secrets.json inside the ephemeral running
container.  So JavaScript, Python and Go programs can load the secrets programatically, if desired.  The combined secrets
file location is exported as $SECRETS_FILE into the running --start, --run and primary command's environments

##### Security Concerns
1. **Reading secrets from files** -- Dockerfy only passes secrets to programs via configuration files to prevent leakage. Secrets could be passed to programs via the environment, but programs use the environment in unpredictable ways, such as logging, or perhaps even dumping their state back to the browser.
2. **Installing Secrets** -- The recommended way to install secrets in production environments is to save them to a tightly protected place on the host and then mount that directory into running docker containers that need secrets. Yes, this is host-level security, but at this point in time, if the host running the docker daemon is not secure, then security has already been compromised.
3. **Tokens** -- Tokens that are revokable, or can be configured to expire, are much safer to use as secrets than long-lived passwords.
4. **Hashed and Salted** --  If passwords must be used, they should be stored only in a salted, and hashed form, never as plain-text or base64 or simply encrypted.  Without salt, passwords can be broken with a dictionary attack

#### Executing Templates
This `--template src:dest` option uses the powerful [go language templating](http://golang.org/pkg/text/template/) capability to substitute environment variables and secret settings directly into the template source and writes the result onto the template destination.

#####Simple Template Substitutions -- an nginx.conf.tmpl

	server {
      location / {
        proxy_pass {{ .Env.PROXY_PASS_URL }};
        proxy_set_header Authorization "Basic {{ .Secret.PROXY_PASSWORD }}";

        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_redirect {{ .Env.PROXY_PASS_URL }} $host;
      }
    }

In the above example, all occurances of the string  {{ .Env.PROXY_PASS_URL }} will be replaced with the value of $PROXY_PASS_URL from the container's environment, and {{ .Secret.PROXY_PASSWORD }} will be replaced with its value, giving the result:

	server {
      location / {
        proxy_pass http://myserver.com;
        proxy_set_header Authorization "Basic a2luZzppc25ha2Vk";

        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_redirect http://myserver.com $host;
      }
    }


Note: $host and $remote_addr are Nginx variable that are set on a per-request basis NOT from the environment.

##### Advanced Templates
But go's templates offer advanced features such as if-statements and comments.

	server {
    {{/* only set up proxy_pass if PROXY_PASS_URL is set in the environment */}}
    {{ if .Env.PROXY_PASS_URL }}
      location / {
        proxy_pass {{ .Env.PROXY_PASS_URL }};

        {{ if .Secret.PROXY_PASSWORD }}
        proxy_set_header Authorization "Basic {{ .Secret.PROXY_PASSWORD }}";
        {{ end }}


        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_redirect {{ .Env.PROXY_PASS_URL }} $host;
      }
    {{ end }}
    }

If your source language uses {{ }} for some other purpose, you can avoid the conflict by using the `--delims` option to specify alternative delimiters such as "<%:%>"

##### Built-in Functions
There are a few built in functions as well:

  * `default $var $default` - Returns a default value for one that does not exist. `{{ default .Env.VERSION "0.1.2" }}`
  * `contains $map $key` - Returns true if a string is within another string
  * `exists $path` - Determines if a file path exists or not. `{{ exists "/etc/default/myapp" }}`
  * `split $string $sep` - Splits a string into an array using a separator string. Alias for [`strings.Split`][go.string.Split]. `{{ split .Env.PATH ":" }}`
  * `replace $string $old $new $count` - Replaces all occurrences of a string within another string. Alias for [`strings.Replace`][go.string.Replace]. `{{ replace .Env.PATH ":" }}`
  * `parseUrl $url` - Parses a URL into it's [protocol, scheme, host, etc. parts][go.url.URL]. Alias for [`url.Parse`][go.url.Parse]
  * `atoi $value` - Parses a string $value into an int. `{{ if (gt (atoi .Env.NUM_THREADS) 1) }}`
  * `add $arg1 $arg` - Performs integer addition. `{{ add (atoi .Env.SHARD_NUM) -1 }}`

##### Secrets in Templates
If you're running in development mode and mounting -v $PWD:/app in your docker container, we recommend:

1. Create a ~/.secrets directory with permissions 700
2. Create a separate secrets file for each application and deployment environment with permissions 600.  Having separate files allows you to avoid lumping all your secrets for all applications and deployment environments into a single file.

	~/.secrets/my-application--production.env
	~/.secrets/my-application--staging.env

3. Export SECRETS_FILES=/secrets/my-application--$DEPLOYMENT_ENV.env
4. Avoid writing templates to your mounted worktree.  **The expanded results might contain secrets!!** and even worse, if you forget to add them to your .gitignore file, then **your secrets could wind up on github.com!!**  Instead, write them to /etc/ or some other place inside the running container that will be forgotten when the container exits.


### Waiting for other dependencies

It is common when using tools like [Docker Compose](https://docs.docker.com/compose/) to depend on services in other linked containers, however oftentimes relying on [links](https://docs.docker.com/compose/compose-file/#links) is not enough - whilst the container itself may have _started_, the _service(s)_ within it may not yet be ready - resulting in shell script hacks to work around race conditions.

**Dockerfy** gives you the ability to wait for services on a specified protocol (`tcp`, `tcp4`, `tcp6`, `http`, and `https`) before running commands, starting services, or starting your application.

NOTE: MySql server is not an HTTP server, so use the tcp protocol instead of http

	$ dockerfy --wait 'tcp://{{ .Env.MYSQLSERVER }}:{{ .Env.MYSQLPORT }}' --timeout 120s ...

You can specify multiple dependancies by repeating the --wait flag.  If the dependancies fail to become available before the timeout (which defaults to 10 seconds), then dockery will exit, and your primary command will not be run.

NOTE: If for some reason dockerfy cannot resolve the DNS names for links try exporting GODEBUG=netdns=cgo to force dockerfy to use cgo for DNS resolution.  This is a known issue on Docker version 1.12.0-rc3, build 91e29e8, experimental for OS X.

### Running Commands
The `--run` option gives you the opportunity to run commands **after** the overlays, secrets and templates have been processed, but **before** the primary program begins.  You can run anything you like, even bash scripts like this:

	$ dockerfy  \
		--run rm -rf /tmp/* -- \
		--run bash -c "sleep 10, echo 'Lets get started now'" -- \
		nginx -g "daemon off;"

All options up to but not including the '--' will be passed to the command.  You can run as many commands as you like, they will be run in the same order as how they were provided on the command line, and all commands must finish **successfully** or **dockerfy** will exit and your primary program will never run.


### Starting Services
The `--start` option gives you the opportunity to start a commands as a service **after** the overlays, secrets and templates have been processed, and all --run commands have completed,  but **before** the primary program begins.  You can start anything you like as a service, even bash scripts like this:

	$ dockerfy  \
		--start "bash -c "while true; do rm -rf /tmp/cache/*; sleep 3600; done" -- \
		nginx -g "daemon off;"

All options up to but not including the '--' will be passed to the command.  You can start as many services as you like, they will all be started in the same order as how they were provided on the command line, and all commands must continue **successfully** or **dockerfy** will
stop your primary command and exit, and the container will stop.

### Switching User Accounts
The `--user` option gives you the ability specify which user accounts with which to run commands or start services.  The `--user` flag takes either a username or UID as its argument, and affects all subsequent commands.

  $ dockerfy \
    --user mark --run id -F -- \
    --user bob  --run id -F -- \
    --user 0    --run id -F -- \
    id -a

The above command will first run the `id -F` command as user "mark", which will print mark's full name "Mark Riggins".
Then it will print bob's full name.  Next it will print the full name of the account with user id 0, which happens to be "root".  Finally the primary command `id` will run with as the user account of the `last` invokation of the `--user` option, giving us the full id information for the root account.

The **dockerfy** command itself will continue to run as the root user so it will have permission to monitor and signal any services that were started.

### Reaping Zombies
Long-lived containers should with services use the `--reap` option to clean up any zombie processes that might arise if a service fails to wait for its child processes to die.  Otherwise, eventually the process table can fill up and your container will become unresponsive.  Normally the init daemon would do this important task, but docker containers do not have an init daemon, so **dockerfy** will assume the responsibility.

Note that in order for this work fully, **dockerfy** should be the primary processes with pid 1. Orphaned child processes are all adopted by the primary process, which allows its to wait for them and collect their exit codes and signals, thus clearing the defunct process table entry.   This means that **dockerfy** must be the FIRST command in your ENTRYPOINT or CMD inside your Dockerfile

### Propagating Signals
**Dockerfy** passes SIGHUP, SIGINT, SIGQUIT, SIGTERM and SIGKILL to all commands and services, giving them a brief chance to respond, and then kills them and exits.  This allows your container to exit gracefully, and completely shut down services, and not hang when it us run in interactive mode via `docker run -it ...` when you type ^C

### Tailing Log Files
Some programs (like nginx) insist on writing their logs to log files instead of stdout and stderr.  Although nginx can be tricked into doing the desired thing by replacing the default log files with symbolic links to /dev/stdout and /dev/stderr, we really don't know how every program out there does its logging, so **dockerfy** gives you to option of tailing as many log files as you wish to stdout and stderr via the --stdout and --stderr flags.

	$ dockerfy --stdout info.log --stdout perf.log


If `inotify` does not work in you container, you use `--log-poll` to poll for file changes instead.



## Installation

Download the latest version in your container:
[releases](https://github.com/markriggins/dockerfy/releases)

For Linux Amd64 Systems:

```
RUN wget https://github.com/markriggins/dockerfy/files/204898/dockerfy-linux-amd64-0.0.4.tar.gz; \
	tar -C /usr/local/bin -xzvf dockerfy-linux-amd64*.gz; \
	rm dockerfy-linux-amd64*.gz;
```
But of course, use the latest release!


##Inspiration and Open Source Usage
Dockerfy is based on the work of others, relying heavily on jwilder's wait, and log tailer, [ dockerize](https://github.com/jwilder/dockerize) and  miekg's  dinit ](https://github.com/miekg/dinit) a small init-like "daemon", and other tips on stackoverflow and other places of public commentary about docker.

The secrets injection, overlays, and commands that run before the primary command starts and the command-line syntax for running commands and starting services are unique to **dockerfy**.

See:
[A Simple Way To Dockerize Applications](http://jasonwilder.com/blog/2014/10/13/a-simple-way-to-dockerize-applications/)




