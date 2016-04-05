.PHONY : build run-staging run-prod stop run-proxy-amazon run-prod-secrets
.PHONY : zombie-maker-debian-binary

build: zombie-maker
	cd ../..; goxc -os linux -arch amd64
	cp -f $(shell bash -xc 'find $(GOPATH)/bin -name dockerfy_linux_amd64.tar.gz| head -1') .dockerfy_linux_amd64.tar.gz
	docker build -t dockerfied-nginx .
	rm .dockerfy_linux_amd64.tar.gz


zombie-maker: zombie-maker.c
	# build a native and debian binary
	docker run --rm -v "$(PWD)":/tmp -w /tmp gcc gcc -o zombie-maker-debian-binary zombie-maker.c
	gcc -o zombie-maker zombie-maker.c


stop:
	@docker rm -f dockerfied-nginx 2>/dev/null || true

run-staging: stop
	docker run -d -p 80:80 -e DEPLOYMENT_ENV=staging --name dockerfied-nginx dockerfied-nginx \
		     -overlay /tmp/overlays/_common/html:/usr/share/nginx/ \
		     -overlay /tmp/overlays/MISSING/html:/usr/share/nginx/ \
	         -overlay "/tmp/overlays/{{ .Env.DEPLOYMENT_ENV }}/html:/usr/share/nginx/" \
	         -- nginx
	open http://$(shell docker-machine ip $(shell docker-machine active))/robots.txt
	open http://$(shell docker-machine ip $(shell docker-machine active))

run-prod: stop
	docker run -d -p 80:80 -e DEPLOYMENT_ENV=prod --name dockerfied-nginx dockerfied-nginx \
		     -overlay /tmp/overlays/_common/html:/usr/share/nginx/ \
	         -overlay "/tmp/overlays/{{ .Env.DEPLOYMENT_ENV }}/html:/usr/share/nginx/" \
	         -- nginx
	open http://$(shell docker-machine ip $(shell docker-machine active))/robots.txt

run-proxy-amazon: stop
	docker run -d -p 80:80 -e PROXY_PASS_URL="https://www.amazon.com/" --name dockerfied-nginx dockerfied-nginx
	open http://$(shell docker-machine ip $(shell docker-machine active))

run-prod-secrets: stop
	docker run -d -p 80:80 -e DEPLOYMENT_ENV=prod --name dockerfied-nginx \
				-v $(PWD):/secrets \
				-e SECRETS_FILE=/secrets/secrets.env \
				dockerfied-nginx \
					-overlay /tmp/overlays/_common/html:/usr/share/nginx/ \
					-overlay "/tmp/overlays/{{ .Env.DEPLOYMENT_ENV }}/html:/usr/share/nginx/" \
					-template "/secrets/secrets.html.tmpl:/usr/share/nginx/html/secrets.html" \
 			        -- nginx 
	open http://$(shell docker-machine ip $(shell docker-machine active))/secrets.html


test: start-tests run-zombie-test run-service-stops-too-soon-test run-primary-stops-while-service-is-running-test

start-tests:
	@echo "\nRunning all tests"
	@echo "--------------------------------------------------------------------------"

run-zombie-test: 
	@echo "\nRunning Zombie Test -- expect 10 zombies to get reaped"
	@echo "--------------------------------------------------------------------------"
	@docker rm -f dockerfied-nginx 2>/dev/null || true
	@echo "\nRun zombie-maker inside dockerfied-nginx"
	@echo "--------------------------------------------------------------------------"
	docker run -d --name dockerfied-nginx dockerfied-nginx -verbose -reap -reap-poll-interval 2s \
		-run zombie-maker 10 1 -- \
		-run sleep 3 -- \
		-run ps -ef -- \
		sleep 120 >/dev/null
	@sleep 5
	@echo "\nPs -ef looking initial zombies"
	@echo "--------------------------------------------------------------------------"
	@docker exec dockerfied-nginx ps -ef | egrep defunct
	
	@sleep 5
	@echo "\nPs -ef looking for leftover zombies -- should be empty"
	@echo "--------------------------------------------------------------------------"
	@docker exec dockerfied-nginx ps -ef | egrep defunct || true

	@echo "\nLogs"
	@echo "--------------------------------------------------------------------------"
	@docker logs dockerfied-nginx 

	@docker logs dockerfied-nginx 2>&1| egrep -q 'Reaped 10 zombies' || (echo FAILED TO REAP; exit 1)

	@echo "run-zombie-test PASSED"
	@docker rm -f dockerfied-nginx 2>/dev/null || true


run-service-stops-too-soon-test: 
	@echo "\nRunning 'run-service-stops-too-soon-test'"
	@echo "\tWhen the service exits, the primary command should also stop"
	@echo "--------------------------------------------------------------------------"
	@docker rm -f dockerfied-nginx 2>/dev/null || true
	docker run -d -it --name dockerfied-nginx dockerfied-nginx -verbose \
		-start bash -c 'sleep 5 && echo "service exiting"' -- \
		sleep 10 >/dev/null
	@sleep 7
	docker logs dockerfied-nginx


run-primary-stops-while-service-is-running-test: 
	@echo "\nRunning 'run-primary-stops-while-service-is-running-test'"
	@echo "\tThis should kill the services before they have a chance to finish"
	@echo "--------------------------------------------------------------------------"
	@docker rm -f dockerfied-nginx 2>/dev/null || true
	docker run -d -it --name dockerfied-nginx dockerfied-nginx -verbose \
		-start bash -c 'sleep 10 && echo "ERROR: service-1 STILL RUNNING"' -- \
		-start bash -c 'sleep 10 && echo "ERROR: service-2 STILL RUNNING"' -- \
		-start bash -c 'sleep 10 && echo "ERROR: service-3 STILL RUNNING"' -- \
		sleep 5 >/dev/null
	@sleep 7
	docker logs dockerfied-nginx
	

run-before-primary-test:
	@echo "\nrun-before-primary-test:"
	@echo "\tBefore the service starts, the -run commands should finish in order"
	@echo "--------------------------------------------------------------------------"
	@docker rm -f dockerfied-nginx 2>/dev/null || true
	docker run -d -it --name dockerfied-nginx dockerfied-nginx -verbose \
		-run bash -c 'sleep 3; echo "1 DONE"' -- \
		-run bash -c 'sleep 2; echo "2 DONE"' -- \
		-run bash -c 'echo "3 DONE"' -- \
		bash -c "sleep 4; echo '4 PRIMARY COMMAND DONE'"
	@sleep 15
	docker logs dockerfied-nginx 2>&1 | egrep '^[0-9].*DONE' 

