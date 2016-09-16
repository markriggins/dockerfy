.PHONY : build run-staging run-prod stop run-proxy-amazon run-prod-secrets
.PHONY : .zombie-maker-debian-binary

TAG:=`git describe --tags`

build: .mk.dockerfy .zombie-maker-debian-binary .mk.nginx-with-dockerfy-and-zombie-maker

.mk.dockerfy:
	cd .. && make dockerfy

.mk.nginx-with-dockerfy-and-zombie-maker:
	cd .. && make nginx-with-dockerfy
	docker build -t  nginx-with-dockerfy-and-zombie-maker .


.zombie-maker-debian-binary: zombie-maker.c
	# build a native and debian binary
	docker run --rm -v "$(PWD)":/tmp -w /tmp gcc gcc -o .zombie-maker-debian-binary zombie-maker.c
	# gcc -o zombie-maker zombie-maker.c


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


test: build just-test

just-test: start-tests run-zombie-test \
	run-service-stops-too-soon-test \
	run-primary-stops-while-service-is-running-test \
	run-before-primary-test run-fails-before-primary-test \
	run-primary-service-exits run-wait-test \
	run-user-option-test run-option-expansion-test \
	run-exit-code-test run-template-funcs-test
	@echo "\n\nALL TESTS PASSED"

start-tests:
	@echo "\nRunning all tests"
	@echo "################################################################################"

run-zombie-test:
	@echo "\n\nRunning Zombie Test -- expect 10 zombies to get reaped"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "\nRun zombie-maker inside test-nginx"
	@echo "--------------------------------------------------------------------------"
	docker run --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose --reap --reap-poll-interval 2s \
		--run zombie-maker 10 1 -- \
		--run sleep 3 -- \
		--run ps -ef -- \
		sleep 10 >/dev/null 2>&1
	@echo "\nLooking for <defunct> processes found by the above 'ps -ef'"
	@echo "--------------------------------------------------------------------------"
	[ `docker logs test-nginx 2>&1 | egrep defunct | wc -l` == 10 ]

	@echo "\nMake sure all 10 zombies got reaped"
	@echo "--------------------------------------------------------------------------"
	@docker logs test-nginx 2>&1| egrep -q 'Reaped 10 zombies' || (echo FAILED TO REAP; exit 1)

	@echo "run-zombie-test PASSED"



run-service-stops-too-soon-test:
	@echo "\n\nRunning 'run-service-stops-too-soon-test'"
	@echo "\tWhen the service exits, the primary command should also stop"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--start bash -c 'sleep 5 && echo "SERVICE EXITED"' -- \
		bash -c 'sleep 10 && echo "PRIMARY COMMAND FINISHED"' >/dev/null 2>&1 || true
	docker logs test-nginx 2>&1 | egrep -q '^SERVICE EXITED'
	docker logs test-nginx 2>&1 | egrep -q '^PRIMARY COMMAND FINISHED' && exit 1 || true

	@echo "run-service-stops-too-soon-test PASSED"


run-primary-stops-while-service-is-running-test:
	@echo "\n\nRunning 'run-primary-stops-while-service-is-running-test'"
	@echo "\tThis should kill the services before they have a chance to finish"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -it -d --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
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
	@echo "\n\nrun-before-primary-test:"
	@echo "\tBefore the service starts, the -run commands should finish in order"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
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
	@echo "\n\nrun-fails-before-primary-test:"
	@echo "\tIf a --run command fails, the primary command should never start"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose --debug \
		--run bash -c 'echo "RUN COMMAND FAILED"; exit 1' -- \
		bash -c "echo 'PRIMARY COMMAND DONE'" >/dev/null 2>&1 || true
	docker logs test-nginx 2>&1 | egrep -q '^RUN COMMAND FAILED'
	docker logs test-nginx 2>&1 | egrep -q '^PRIMARY COMMAND DONE' && exit 1 || true

	@echo "run-before-primary-test PASSED"

run-primary-service-exits:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "\n\nrun-primary-service-exits:"
	@echo "\tVerify that when the primary service finishes, the container exits"
	@echo "################################################################################"
	docker run -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose echo DONE
	docker inspect --format '{{ .State.Status }}' test-nginx | egrep -q exited
	docker logs test-nginx 2>&1 | egrep -q '^DONE'

	@echo "run-primary-service-exits PASSED"


run-wait-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "\n\nrun-wait-test:"
	@echo "\tVerify that when the primary command runs after we wait for something"
	@echo "################################################################################"
	docker run -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--wait http://amazon.com \
		echo DONE
	docker logs test-nginx 2>&1 | egrep -q 'Waiting for host: http://amazon.com'
	docker logs test-nginx 2>&1 | egrep -q '^DONE'
	@echo "run-wait-test PASSED"

run-user-option-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "\n\nrun-user-option-test:"
	@echo "\tVerify that the --user option affects the user account of subsequent commands"
	@echo "################################################################################"
	docker run -it -e SECRETS_FILES=/secrets/secrets.json:/secrets/secrets.2.json \
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

	@echo "\n\nrun-option-expansion-test:"
	@echo "\tVerify that {{ .Env.VARS }} are expanded before use by dockerfy and \$$VARS are NOT"
	@echo "################################################################################"
	docker run -it -e SECRETS_FILES=/secrets/secrets.json \
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
	@echo "\n\nrun-exit-code-test:"
	@echo "\tVerify that dockerfy exits with the exit_code of the failing command"
	@echo "################################################################################"
	docker run -it --rm nginx-with-dockerfy-and-zombie-maker --verbose \
		-- bash -c 'exit 5' >/dev/null 2>&1 || [ $$? == 5 ]
	docker run -it --rm nginx-with-dockerfy-and-zombie-maker --verbose \
		--run bash -c 'exit 3' -- bash -c 'exit 5' >/dev/null 2>&1 || [ $$? == 3 ]
	docker run -it --rm nginx-with-dockerfy-and-zombie-maker --verbose \
		--start bash -c 'sleep 2; exit 3' -- bash -c 'exit 5' >/dev/null 2>&1 || [ $$? == 5 ]
	@echo "run-exit-code-test PASSED"


run-trim-args-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true
	@echo "\n\nrun-trim-args-test:"
	@echo "\tVerify that dockerfy trims its own args, but not those to --run or --start cmds"
	@echo "################################################################################"
	docker run -it --name test-nginx \
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
	@echo "\n\nrun-template-funcs-test:"
	@echo "\tVerify that dockerfy template functions work"
	@echo "################################################################################"
	@echo "\tgetenv and concat"
	@dockerfy -- echo '{{ concat "P" "W" "D" | getenv }}' | egrep -q "^$(PWD)$$"
	@echo "\tN"
	N=20 dockerfy -- echo '{{ $$N := getenv "N"}}{{range $$i, $$v := sequence "0" $$N}}{{$$v}}{{end}}' | egrep  -q '^012345678910111213141516171819$$'
	@echo "run-template-funcs-test PASSED"
