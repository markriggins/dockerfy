.PHONY : build run-staging run-prod stop run-proxy-amazon run-prod-secrets
.PHONY : .zombie-maker-debian-binary

SHELL:=/bin/bash
TAG:=`git describe --tags`

export SHELL = /bin/bash

build: ../dockerfy .mk.nginx-with-dockerfy-and-zombie-maker

clean:
	rm -f .mk* .zombie-maker-debian-binary


../dockerfy:
	cd .. && make dockerfy


.mk.nginx-with-dockerfy-and-zombie-maker: Dockerfile.nginx-with-dockerfy-and-zombie-maker .zombie-maker-debian-binary
	cd .. && make nginx-with-dockerfy
	docker build --file Dockerfile.nginx-with-dockerfy-and-zombie-maker -t  nginx-with-dockerfy-and-zombie-maker .


.zombie-maker-debian-binary: zombie-maker.c
	# build a debian binary that can run in an nginx container
	docker run --rm -v "$(PWD)":/tmp -w /tmp gcc gcc -o .zombie-maker-debian-binary zombie-maker.c


stop:
	@docker rm -f test-nginx >/dev/null 2>&1 || true


run-staging: stop
	docker run -d -p 80:80 -e DEPLOYMENT_ENV=staging --name test-nginx test-nginx \
		     --overlay /tmp/overlays/_common/html:/usr/share/nginx/ \
		     --overlay /tmp/overlays/MISSING/html:/usr/share/nginx/ \
	         --overlay '/tmp/overlays/{{ .Env.DEPLOYMENT_ENV }}/html:/usr/share/nginx/' \
	         -- nginx -g "daemon off;"

	open http://$(shell docker-machine ip $(shell docker-machine active))/robots.txt
	open http://$(shell docker-machine ip $(shell docker-machine active))


run-prod: stop
	docker run -d -p 80:80 -e DEPLOYMENT_ENV=prod --name test-nginx nginx-with-dockerfy-and-zombie-maker \
		     --overlay /tmp/overlays/_common/html:/usr/share/nginx/ \
	         --overlay '/tmp/overlays/{{ .Env.DEPLOYMENT_ENV }}/html:/usr/share/nginx/' \
	         -- nginx -g "daemon off;"
	open http://$(shell docker-machine ip $(shell docker-machine active))/robots.txt


run-proxy-amazon: stop
	docker run -d -p 80:80 -e PROXY_PASS_URL="https://www.amazon.com/" --name test-nginx nginx-with-dockerfy-and-zombie-maker
	open http://$(shell docker-machine ip $(shell docker-machine active))


run-prod-secrets: stop
	docker run -d -p 80:80 -e DEPLOYMENT_ENV=prod --name test-nginx \
				-v $(PWD):/secrets \
				-e SECRETS_FILES=/secrets/secrets.env \
				nginx-with-dockerfy-and-zombie-maker \
					--verbose \
					--overlay /tmp/overlays/_common/:/usr/share/nginx/ \
					--overlay '/tmp/overlays/{{ .Env.DEPLOYMENT_ENV }}/html:/usr/share/nginx/' \
					--template '/secrets/secrets.html.tmpl:/usr/share/nginx/html/secrets.html' \
					--secrets-files "/secrets/secrets.json:/secrets/secrets.2.json" \
 			        -- nginx -g "daemon off;"
	open http://$(shell docker-machine ip $(shell docker-machine active))/secrets.html


#########################################################################
#                     UNIT TESTS                                        #
#########################################################################

tmpfile := $(shell echo "/tmp/$$RANDOM.out")

test: build
	@{ \
		if ! make just-test; then \
			echo "DOCKER LOGS FROM LAST text-nginx CONTAINER:"; \
			docker logs test-nginx 2>&1; \
			exit 1; \
		fi; \
	}
	@rm -f $(tmpfile)


just-test: start-tests run-zombie-test \
	run-service-stops-too-soon-test \
	run-primary-stops-while-service-is-running-test \
	run-before-primary-test run-fails-before-primary-test \
	run-primary-service-exits run-wait-test \
	run-user-option-test run-option-expansion-test \
	run-exit-code-test run-template-funcs-test \
	run-signal-passing-test

	@echo -e "\n\nALL TESTS PASSED"


start-tests:
	@echo -e "\nRunning all tests"
	@echo "################################################################################"


run-zombie-test:
	@echo -e "\n\nRunning Zombie Test -- expect 10 zombies to get reaped"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo -e "\nRun zombie-maker inside test-nginx"
	@echo "--------------------------------------------------------------------------"
	docker run --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose --reap --reap-poll-interval 2s \
		--run zombie-maker 10 1 -- \
		--run sleep 3 -- \
		--run ps -ef -- \
		sleep 10 >/dev/null 2>&1
	@echo -e "\nLooking for <defunct> processes found by the above 'ps -ef'"
	@echo "--------------------------------------------------------------------------"
	[ `docker logs test-nginx 2>&1 | egrep defunct | wc -l` == 10 ]

	@echo -e "\nMake sure all 10 zombies got reaped"
	@echo "--------------------------------------------------------------------------"
	@docker logs test-nginx 2>&1| egrep -q 'Reaped 10 zombies' || (echo FAILED TO REAP; exit 1)

	@echo "run-zombie-test PASSED"



run-service-stops-too-soon-test:
	@echo -e "\n\nRunning 'run-service-stops-too-soon-test'"
	@echo -e "\tWhen the service exits, the primary command should also stop"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	#docker run --rm --entrypoint bash nginx-with-dockerfy-and-zombie-maker -c 'ls -la /secrets {{.Env.HOME}}/.secret* ||true'
	docker run --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--start bash -c 'sleep 5 && echo "SERVICE EXITED"' -- \
		bash -c 'sleep 10 && echo "PRIMARY COMMAND FINISHED"' >/dev/null 2>&1 || true
	docker logs test-nginx 2>&1 | egrep -q '^SERVICE EXITED'
	docker logs test-nginx 2>&1 | egrep -q '^PRIMARY COMMAND FINISHED' && exit 1 || true

	@echo "run-service-stops-too-soon-test PASSED"


run-primary-stops-while-service-is-running-test:
	@echo -e "\n\nRunning 'run-primary-stops-while-service-is-running-test'"
	@echo -e "\tThis should kill the services before they have a chance to finish"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -d --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--start bash -c 'sleep 60 && echo "ERROR: service-1 STILL RUNNING"' -- \
		--start bash -c 'sleep 60 && echo "ERROR: service-2 STILL RUNNING"' -- \
		--start bash -c 'sleep 60 && echo "ERROR: service-3 STILL RUNNING"' -- \
		bash -c 'sleep 1; echo "PRIMARY DONE"'
	@sleep 5
	[ `docker inspect --format '{{.State.Status}}' test-nginx` == 'exited' ]
	[ `docker logs test-nginx 2>&1  | egrep -c '^ERROR:.*STILL RUNNING'` == 0 ]
	@docker logs test-nginx 2>&1  | egrep -q 'Primary Command.*finished'
	@docker logs test-nginx 2>&1  | egrep -q 'PRIMARY DONE'
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "run-primary-stops-while-service-is-running-test PASSED"


run-before-primary-test:
	@echo -e "\n\nrun-before-primary-test:"
	@echo -e "\tBefore the service starts, the -run commands should finish in order"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--run bash -c 'sleep 3; echo "1 DONE"' -- \
		--run bash -c 'sleep 2; echo "2 DONE"' -- \
		--run bash -c 'echo "3 DONE"' -- \
		bash -c "sleep 4; echo '4 PRIMARY COMMAND DONE'" >/dev/null 2>&1
	docker logs test-nginx 2>&1 | egrep -q '^1 DONE'
	docker logs test-nginx 2>&1 | egrep -q '^2 DONE'
	docker logs test-nginx 2>&1 | egrep -q '^3 DONE'
	docker logs test-nginx 2>&1 | egrep -q '^4 PRIMARY COMMAND DONE'

	@echo "run-before-primary-test PASSED"


run-fails-before-primary-test:
	@echo -e "\n\nrun-fails-before-primary-test:"
	@echo -e "\tIf a --run command fails, the primary command should never start"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose --debug \
		--run bash -c 'echo "RUN COMMAND FAILED"; exit 1' -- \
		bash -c "echo 'PRIMARY COMMAND DONE'" >/dev/null 2>&1 || true
	docker logs test-nginx 2>&1 | egrep -q '^RUN COMMAND FAILED'
	docker logs test-nginx 2>&1 | egrep -q '^PRIMARY COMMAND DONE' && exit 1 || true

	@echo "run-before-primary-test PASSED"


run-primary-service-exits:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo -e "\n\nrun-primary-service-exits:"
	@echo -e "\tVerify that when the primary service finishes, the container exits"
	@echo "################################################################################"
	docker run --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose echo DONE
	docker inspect --format '{{ .State.Status }}' test-nginx | egrep -q exited
	docker logs test-nginx 2>&1 | egrep -q '^DONE'

	@echo "run-primary-service-exits PASSED"


run-wait-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo -e "\n\nrun-wait-test:"
	@echo -e "\tVerify that when the primary command runs after we wait for something"
	@echo "################################################################################"
	docker run --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--wait http://amazon.com \
		echo DONE
	docker logs test-nginx 2>&1 | egrep -q 'Waiting for host: http://amazon.com'
	docker logs test-nginx 2>&1 | egrep -q '^DONE'
	@echo "run-wait-test PASSED"


run-wait-timeout-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo -e "\n\nrun-wait-timeout-test:"
	@echo -e "\tVerify that when the primary command does not run if --wait times out"
	@echo "################################################################################"
	docker run --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
	    --timeout 2s \
		--wait http://amazon.com:81 \
		echo DONE && exit 1 || true
	docker logs test-nginx 2>&1 | egrep -q 'Waiting for host: http://amazon.com:81'
	docker logs test-nginx 2>&1 | egrep -q 'Timeout after 2s waiting.*amazon'
	docker logs test-nginx 2>&1 | egrep -q '^DONE' && exit 1 || true
	@echo "run-wait-timeout-test PASSED"


run-user-option-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo -e "\n\nrun-user-option-test:"
	@echo -e "\tVerify that the --user option affects the user account of subsequent commands"
	@echo "################################################################################"
	docker run -e SECRETS_FILES=/secrets/secrets.json:/secrets/secrets.2.json \
	    --name test-nginx nginx-with-dockerfy-and-zombie-maker \
	    --debug \
		\
		--user mail \
			--run echo -n "MAIL:" -- \
		 	--run id -a -- \
			--run ls -l '/var/spool/mail/.secrets' -- \
			--run cat '/var/spool/mail/.secrets/combined_secrets.json' -- \
			--run cat '/var/spool/mail/.secrets/secrets.json' -- \
			--run /usr/bin/env -- \
		\
		--user root \
			--run echo -n "ROOT:" -- \
			--run id -a -- \
			--run /usr/bin/env -- \
			--run echo -n "PRIMARY:" -- \
			id -a >/dev/null 2>&1
	docker logs test-nginx 2>/dev/null| egrep -q -- '\-r\-* .*secrets.json'

	docker logs test-nginx 2>/dev/null| fgrep -q -- 'SECRETS_FILE=/var/mail/.secrets/combined_secrets.json'
	docker logs test-nginx 2>/dev/null| fgrep -q -- 'SECRETS_FILES=/var/mail/.secrets/secrets.json:/var/mail/.secrets/secrets.2.json'
	docker logs test-nginx 2>/dev/null| fgrep -q -- 'uid=0(root) gid=0(root) groups=0(root)'
	docker logs test-nginx 2>/dev/null| fgrep -q -- 'uid=8(mail) gid=8(mail) groups=8(mail)'
	docker logs test-nginx 2>/dev/null| fgrep -q -- '"JSON_SECRET": "Jason Voorhees did it"'
	[ $$(docker logs test-nginx 2>/dev/null| fgrep -- 'uid=0(root) gid=0(root) groups=0(root)' | wc -l) == 2 ]
	# '
	@echo "run-user-option-test PASSED"


run-option-expansion-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo -e "\n\nrun-option-expansion-test:"
	@echo -e "\tVerify that {{ .Env.VARS }} are expanded before use by dockerfy and \$$VARS are NOT"
	@echo "################################################################################"
	docker run -e SECRETS_FILES=/secrets/secrets.json \
		-e DB_NAME="My-Db" -e DB_HOST=localhost  \
		-e SC_ENV=local \
	    --name test-nginx nginx-with-dockerfy-and-zombie-maker \
	    --verbose \
		--secrets-files '/secrets/secrets.2.json:/secrets/secrets.{{ .Env.SC_ENV }}.json' \
	    --run echo 'DB_HOST={{ .Env.DB_HOST }}' -- \
		--run echo 'DB_NAME_XP_1=$$DB_NAME' -- \
		--run echo 'DB_NAME_XP_2={{ .Env.DB_NAME }}' --  \
		--run echo 'AnotherSecret={{ .Secret.AnotherSecret }}' -- \
		--run echo 'HOW_TO_FIND_A_GOOD_RESTAURANT={{ .Secret.HOW_TO_FIND_A_GOOD_RESTAURANT }}' -- \
		echo 'JSON_SECRET={{ .Secret.JSON_SECRET }}'

	docker logs test-nginx 2>/dev/null
	docker logs test-nginx 2>/dev/null | fgrep -q -- 'DB_HOST=localhost'
	docker logs test-nginx 2>/dev/null | fgrep -q -- 'JSON_SECRET=Jason Voorhees did it'
	docker logs test-nginx 2>/dev/null | fgrep -q -- 'AnotherSecret=Jason Voorhees is still alive'
	docker logs test-nginx 2>/dev/null | fgrep -q -- 'DB_NAME_XP_1=$$DB_NAME'
	docker logs test-nginx 2>/dev/null | fgrep -q -- 'DB_NAME_XP_2=My-Db'
	docker logs test-nginx 2>/dev/null | fgrep -q -- 'HOW_TO_FIND_A_GOOD_RESTAURANT=Ask a local'

	# @docker rm -f test-nginx >/dev/null 2>&1 || true
	@echo "run-option-expansion-test PASSED"


run-exit-code-test:
	@echo -e "\n\nrun-exit-code-test:"
	@echo -e "\tVerify that dockerfy exits with the exit_code of the failing command"
	@echo "################################################################################"
	docker run --rm nginx-with-dockerfy-and-zombie-maker --verbose \
		-- bash -c 'exit 5' >/dev/null 2>&1 || [ $$? == 5 ]
	docker run --rm nginx-with-dockerfy-and-zombie-maker --verbose \
		--run bash -c 'exit 3' -- bash -c 'exit 5' >/dev/null 2>&1 || [ $$? == 3 ]
	docker run --rm nginx-with-dockerfy-and-zombie-maker --verbose \
		--start bash -c 'sleep 2; exit 3' -- bash -c 'exit 5' >/dev/null 2>&1 || [ $$? == 5 ]
	@echo "run-exit-code-test PASSED"


run-exit-code-test-with-signals:
	#
	#  NOTE:  this test fails so we can't run it.  When a `sleep` is interrupted
	#         it exits with code 130.  But the golang syscalls for this report -1
	#
	@echo -e "\n\nrun-exit-code-test-with-signals:"
	@echo -e "\tVerify that dockerfy exits with the exit_code of the failing command"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -d --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose -- sleep 45678
	docker exec test-nginx ps -e --format "pid args"
	docker exec test-nginx bash -c 'ps -e --format "pid args" | egrep "sleep 45678" | egrep -v grep | egrep -v dockerfy | while read pid rest; do kill -2 $$pid; done'
	sleep 2

	docker logs test-nginx
	[ $$(docker inspect --format '{{.State.ExitCode}}' test-nginx) == 130 ] || \
		( echo "Wrong exit code $$(docker inspect --format '{{.State.ExitCode}}' test-nginx) != 130 "; exit 1)


run-trim-args-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true
	@echo -e "\n\nrun-trim-args-test:"
	@echo -e "\tVerify that dockerfy trims its own args, but not those to --run or --start cmds"
	@echo "################################################################################"
	docker run --name test-nginx \
		-v $(PWD):/secrets  \
		nginx-with-dockerfy-and-zombie-maker  \
		' --debug ' \
		' --secrets-files ' " /secrets/secrets.json:/secrets/secrets.2.json " \
		--run echo '  LEADING SPACES   ' -- echo DONE >/dev/null 2>&1
	docker logs test-nginx 2>&1 | egrep -q 'debugging output'
	docker logs test-nginx 2>&1 | egrep -q '^  LEADING SPACES'
	docker logs test-nginx 2>&1 | egrep -q 'loaded secret: JSON_SECRET'
	@echo "run-trim-args-test PASSED"


run-template-funcs-test:
	@echo -e "\n\nrun-template-funcs-test:"
	@echo -e "\tVerify that dockerfy template functions work"
	@echo "################################################################################"
	@echo -e "\tgetenv and concat"
	@../dockerfy -- echo '{{ concat "P" "W" "D" | getenv }}' | egrep -q "^$(PWD)$$"
	@echo -e "\tsequence"
	@N=20 ../dockerfy -- echo '{{ $$N := getenv "N"}}{{range $$i, $$v := sequence "1" $$N}}{{$$v}},{{end}}' \
		| egrep -q '^1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,$$'
	@echo -e "\tsplit"
	@../dockerfy -- echo '{{range $$i, $$v := split "a,b,c" ","}}{{$$v}}__{{end}}' | egrep  -q '^a__b__c__$$'
	@echo "run-template-funcs-test PASSED"


run-signal-passing-test:
	@echo -e "\n\nrun-signal-passing-test: "
	@echo -e "\tVerify that dockerfy passes signals to start commands and the primary command"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -d --name test-nginx nginx-with-dockerfy-and-zombie-maker --debug \
		--run echo run-signal-passing-test -- \
		--start nohup sleep 500 -- \
		--start nohup sleep 600 -- \
		nohup sleep 700

	@echo -e "\n\tverify that SIGHUP passes thru to children"
	@# Send a SIGHUP, which will propagate to the children, but NOT cause them to terminate
	@# because they are nohup'ed.   NOTE: dockerfy is always PID 1
	@docker exec test-nginx kill -s SIGHUP 1
	@sleep 1
	docker logs test-nginx 2>&1 | egrep -q 'Command `nohup sleep 500` received signal hangup'
	docker logs test-nginx 2>&1 | egrep -q 'Command `nohup sleep 600` received signal hangup'
	docker logs test-nginx 2>&1 | egrep -q 'Command `nohup sleep 700` received signal hangup'


	@echo -e "\tverify that SIGTERM terminates children"
	@# Send a SIGTERM, which will propagate to the children, AND cause them to terminate
	@docker exec test-nginx kill -s SIGTERM 1
	@sleep 1

	docker logs test-nginx 2>&1 | egrep -q 'Command `nohup sleep 500` exited with error: signal: terminated'
	docker logs test-nginx 2>&1 | egrep -q 'Command `nohup sleep 600` exited with error: signal: terminated'
	docker logs test-nginx 2>&1 | egrep -q 'Command `nohup sleep 700` exited with error: signal: terminated'

	[ $$(docker inspect --format '{{.State.Status}}' test-nginx) == "exited" ] || \
		( echo "dockerfy failed to exit on SIGTERM"; exit 1)

	@echo "run-signal-passing-test PASSED"
