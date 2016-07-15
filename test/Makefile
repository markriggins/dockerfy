.PHONY : build run-staging run-prod stop run-proxy-amazon run-prod-secrets
.PHONY : .zombie-maker-debian-binary

TAG:=`git describe --tags`

build: .zombie-maker-debian-binary .mk.nginx-with-dockerfy-and-zombie-maker


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

just-test: start-tests run-zombie-test run-service-stops-too-soon-test run-primary-stops-while-service-is-running-test \
		run-before-primary-test run-fails-before-primary-test run-primary-service-exits run-wait-test run-user-option-test run-option-expansion-test run-exit-code-test
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
	docker run -d --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose --reap --reap-poll-interval 2s \
		--run zombie-maker 10 1 -- \
		--run sleep 3 -- \
		--run ps -ef -- \
		sleep 120 >/dev/null
	@sleep 5
	@echo "\nPs -ef looking initial zombies"
	@echo "--------------------------------------------------------------------------"
	docker exec test-nginx ps -ef | egrep defunct

	@sleep 5
	@echo "\nPs -ef looking for leftover zombies -- should be empty"
	@echo "--------------------------------------------------------------------------"
	docker exec test-nginx ps -ef | egrep defunct || true
	@docker exec test-nginx ps -ef | egrep -q defunct  && exit 1 || true

	@echo "\nMake sure all 10 zombies got reaped"
	@echo "--------------------------------------------------------------------------"
	@docker logs test-nginx 2>&1| egrep -q 'Reaped 10 zombies' || (echo FAILED TO REAP; exit 1)

	@echo "run-zombie-test PASSED"
	@docker rm -f test-nginx >/dev/null 2>&1 || true



run-service-stops-too-soon-test:
	@echo "\n\nRunning 'run-service-stops-too-soon-test'"
	@echo "\tWhen the service exits, the primary command should also stop"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -d -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--start bash -c 'sleep 5 && echo "service exiting"' -- \
		sleep 10 >/dev/null
	@sleep 7
	docker logs test-nginx 2>&1 | egrep -q 'Service.*stopped'
	@echo "\t Service Stopped"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "run-service-stops-too-soon-test PASSED"


run-primary-stops-while-service-is-running-test:
	@echo "\n\nRunning 'run-primary-stops-while-service-is-running-test'"
	@echo "\tThis should kill the services before they have a chance to finish"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -d -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--start bash -c 'sleep 60 && echo "ERROR: service-1 STILL RUNNING"' -- \
		--start bash -c 'sleep 60 && echo "ERROR: service-2 STILL RUNNING"' -- \
		--start bash -c 'sleep 60 && echo "ERROR: service-3 STILL RUNNING"' -- \
		bash -c 'sleep 1; echo "PRIMARY DONE"' >/dev/null
	@sleep 10
	[ `docker inspect --format '{{.State.Status}}' test-nginx` == 'exited' ]
	[ `docker logs test-nginx 2>&1  | egrep -c '^ERROR:.*STILL RUNNING'` == 0 ]
	@docker logs test-nginx 2>&1  | egrep -q 'Primary Command.*stopped'
	@docker logs test-nginx 2>&1  | egrep -q 'PRIMARY DONE'

	@echo "run-primary-stops-while-service-is-running-test PASSED"


run-before-primary-test:
	@echo "\n\nrun-before-primary-test:"
	@echo "\tBefore the service starts, the -run commands should finish in order"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -d -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--run bash -c 'sleep 3; echo "1 DONE"' -- \
		--run bash -c 'sleep 2; echo "2 DONE"' -- \
		--run bash -c 'echo "3 DONE"' -- \
		bash -c "sleep 4; echo '4 PRIMARY COMMAND DONE'"
	@sleep 15
	docker logs test-nginx 2>&1 | egrep -q '^1 DONE'
	docker logs test-nginx 2>&1 | egrep -q '^2 DONE'
	docker logs test-nginx 2>&1 | egrep -q '^3 DONE'
	docker logs test-nginx 2>&1 | egrep -q '^4 PRIMARY COMMAND DONE'
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "run-before-primary-test PASSED"


run-fails-before-primary-test:
	@echo "\n\nrun-fails-before-primary-test:"
	@echo "\tIf a --run command fails, the primary command should never start"
	@echo "################################################################################"
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	docker run -d -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--run bash -c 'echo "RUN COMMAND FAILED"; exit 1' -- \
		bash -c "echo 'PRIMARY COMMAND DONE'"
	@sleep 5
	docker logs test-nginx 2>&1 | egrep -q '^RUN COMMAND FAILED'
	docker logs test-nginx 2>&1 | egrep -q '^PRIMARY COMMAND DONE' && exit 1 || true
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "run-before-primary-test PASSED"

run-primary-service-exits:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "\n\nrun-primary-service-exits:"
	@echo "\tVerify that when the primary service finishes, the container exits"
	@echo "################################################################################"
	docker run -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose echo DONE
	docker inspect --format '{{ .State.Status }}' test-nginx | egrep -q exited
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "run-primary-service-exits PASSED"


run-wait-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "\n\nrun-wait-test:"
	@echo "\tVerify that when the primary command runs after we wait for something"
	@echo "################################################################################"
	docker run -it --name test-nginx nginx-with-dockerfy-and-zombie-maker --verbose \
		--wait http://amazon.com \
		echo DONE
	docker logs test-nginx 2>&1 | egrep -q '^DONE'
	@echo "run-wait-test PASSED"

run-user-option-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "\n\nrun-user-option-test:"
	@echo "\tVerify that the --user option affects the user account of subsequent commands"
	@echo "################################################################################"
	docker run -d -it -e SECRETS_FILES=/secrets/secrets.json:/secrets/secrets.2.json \
	    --name test-nginx nginx-with-dockerfy-and-zombie-maker \
		--run echo -n "MAIL:" -- --user mail  --run id -a -- \
		--run ls -l '/var/spool/mail/.secrets' -- \
		--run cat '/var/spool/mail/.secrets/combined_secrets.json' -- \
		--run cat '/var/spool/mail/.secrets/secrets.json' -- \
		--run /usr/bin/env -- \
		--run echo -n "ROOT:" -- --user root  --run id -a -- \
		--run /usr/bin/env -- \
		--run echo -n "PRIMARY:" -- \
		id -a
	@sleep 5
	docker logs test-nginx 2>/dev/null| egrep -q -- '\-r\-* .*secrets.json'

	docker logs test-nginx 2>/dev/null| fgrep -q -- 'SECRETS_FILE=/var/mail/.secrets/combined_secrets.json'
	docker logs test-nginx 2>/dev/null| fgrep -q -- 'SECRETS_FILES=/var/mail/.secrets/secrets.json:/var/mail/.secrets/secrets.2.json'
	docker logs test-nginx 2>/dev/null| fgrep -q -- 'uid=0(root) gid=0(root) groups=0(root)'
	docker logs test-nginx 2>/dev/null| fgrep -q -- 'uid=8(mail) gid=8(mail) groups=8(mail)'
	docker logs test-nginx 2>/dev/null| fgrep -q -- '"JSON_SECRET": "Jason Voorhees did it"'
	[ $$(docker logs test-nginx 2>/dev/null| fgrep -- 'uid=0(root) gid=0(root) groups=0(root)' | wc -l) == 2 ]
	# '
	@docker rm -f test-nginx >/dev/null 2>&1 || true
	@echo "run-user-option-test PASSED"

run-option-expansion-test:
	@docker rm -f test-nginx >/dev/null 2>&1 || true

	@echo "\n\nrun-option-expansion-test:"
	@echo "\tVerify that {{ .Env.VARS }} are expanded before use by dockerfy and \$$VARS are NOT"
	@echo "################################################################################"
	docker run -d -it -e SECRETS_FILES=/secrets/secrets.json \
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


	@sleep 5
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
		-- bash -c 'exit 5' || [ $$? == 5 ]
	docker run -it --rm nginx-with-dockerfy-and-zombie-maker --verbose \
		--run bash -c 'exit 3' -- bash -c 'exit 5' || [ $$? == 3 ]
	docker run -it --rm nginx-with-dockerfy-and-zombie-maker --verbose \
		--start bash -c 'sleep 2; exit 3' -- bash -c 'exit 5' || [ $$? == 5 ]
	@echo "run-exit-code-test PASSED"

