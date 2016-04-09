.PHONY : build run-staging run-prod stop run-proxy-amazon run-prod-secrets
.PHONY : zombie-maker-debian-binary

TAG:=`git describe --abbrev=0 --tags`

build: zombie-maker .mk.nginx-with-dockerfy
	cd ../..; goxc -os linux -arch amd64
	docker build -t dockerfied-nginx .

push: .mk.nginx-with-dockerfy
	docker push markriggins/nginx-with-dockerfy

.mk.nginx-with-dockerfy: Dockerfile.nginx-with-dockerfy
	cp -f $(shell bash -xc 'find $(GOPATH)/bin -name dockerfy_linux_amd64.tar.gz| head -1') .dockerfy_linux_amd64.tar.gz
	docker build --file Dockerfile.nginx-with-dockerfy -t markriggins/nginx-with-dockerfy .
	docker tag markriggins/nginx-with-dockerfy markriggins/nginx-with-dockerfy:$(TAG)


zombie-maker: zombie-maker.c
	# build a native and debian binary
	docker run --rm -v "$(PWD)":/tmp -w /tmp gcc gcc -o zombie-maker-debian-binary zombie-maker.c
	gcc -o zombie-maker zombie-maker.c


stop:
	@docker rm -f dockerfied-nginx 2>/dev/null || true

run-staging: stop
	docker run -d -p 80:80 -e DEPLOYMENT_ENV=staging --name dockerfied-nginx dockerfied-nginx \
		     --overlay /tmp/overlays/_common/html:/usr/share/nginx/ \
		     --overlay /tmp/overlays/MISSING/html:/usr/share/nginx/ \
	         --overlay '/tmp/overlays/$$DEPLOYMENT_ENV/html:/usr/share/nginx/' \
	         -- nginx
	open http://$(shell docker-machine ip $(shell docker-machine active))/robots.txt
	open http://$(shell docker-machine ip $(shell docker-machine active))

run-prod: stop
	docker run -d -p 80:80 -e DEPLOYMENT_ENV=prod --name dockerfied-nginx dockerfied-nginx \
		     --overlay /tmp/overlays/_common/html:/usr/share/nginx/ \
	         --overlay '/tmp/overlays/$$DEPLOYMENT_ENV/html:/usr/share/nginx/' \
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
					--verbose \
					--overlay /tmp/overlays/_common/:/usr/share/nginx/ \
					--overlay '/tmp/overlays/{{ .Env.DEPLOYMENT_ENV }}/html:/usr/share/nginx/' \
					--template '/secrets/secrets.html.tmpl:/usr/share/nginx/html/secrets.html' \
					--secrets "/secrets/secrets.json" \
 			        -- nginx 
	open http://$(shell docker-machine ip $(shell docker-machine active))/secrets.html


test: start-tests run-zombie-test run-service-stops-too-soon-test run-primary-stops-while-service-is-running-test run-before-primary-test run-primary-service-exits
	@echo "\n\nALL TESTS PASSED"

start-tests:
	@echo "\nRunning all tests"
	@echo "################################################################################"

run-zombie-test: 
	@echo "\n\nRunning Zombie Test -- expect 10 zombies to get reaped"
	@echo "################################################################################"
	@docker rm -f dockerfied-nginx 2>/dev/null || true
	@echo "\nRun zombie-maker inside dockerfied-nginx"
	@echo "--------------------------------------------------------------------------"
	docker run -d --name dockerfied-nginx dockerfied-nginx --verbose --reap --reap-poll-interval 2s \
		--run zombie-maker 10 1 -- \
		--run sleep 3 -- \
		--run ps -ef -- \
		sleep 120 >/dev/null
	@sleep 5
	@echo "\nPs -ef looking initial zombies"
	@echo "--------------------------------------------------------------------------"
	docker exec dockerfied-nginx ps -ef | egrep defunct
	
	@sleep 5
	@echo "\nPs -ef looking for leftover zombies -- should be empty"
	@echo "--------------------------------------------------------------------------"
	docker exec dockerfied-nginx ps -ef | egrep defunct || true
	@docker exec dockerfied-nginx ps -ef | egrep -q defunct  && exit 1 || true

	@echo "\nLogs"
	@echo "--------------------------------------------------------------------------"
	@docker logs dockerfied-nginx 

	@docker logs dockerfied-nginx 2>&1| egrep -q 'Reaped 10 zombies' || (echo FAILED TO REAP; exit 1)

	@echo "run-zombie-test PASSED"
	@docker rm -f dockerfied-nginx 2>/dev/null || true


run-service-stops-too-soon-test: 
	@echo "\n\nRunning 'run-service-stops-too-soon-test'"
	@echo "\tWhen the service exits, the primary command should also stop"
	@echo "################################################################################"
	@docker rm -f dockerfied-nginx 2>/dev/null || true
	docker run -d -it --name dockerfied-nginx dockerfied-nginx --verbose \
		--start bash -c 'sleep 5 && echo "service exiting"' -- \
		sleep 10 >/dev/null
	@sleep 7
	docker logs dockerfied-nginx 2>&1 | egrep -q 'Service.*stopped' 
	@echo "\t Service Stopped"
	@echo "run-service-stops-too-soon-test PASSED"


run-primary-stops-while-service-is-running-test: 
	@echo "\n\nRunning 'run-primary-stops-while-service-is-running-test'"
	@echo "\tThis should kill the services before they have a chance to finish"
	@echo "################################################################################"
	@docker rm -f dockerfied-nginx 2>/dev/null || true
	docker run -d -it --name dockerfied-nginx dockerfied-nginx --verbose \
		--start bash -c 'sleep 10 && echo "ERROR: service-1 STILL RUNNING"' -- \
		--start bash -c 'sleep 10 && echo "ERROR: service-2 STILL RUNNING"' -- \
		--start bash -c 'sleep 10 && echo "ERROR: service-3 STILL RUNNING"' -- \
		sleep 5 >/dev/null
	@sleep 7
	@docker logs dockerfied-nginx 2>&1  | egrep -q 'Primary Command.*stopped'
	@docker logs dockerfied-nginx 2>&1  | egrep -q '^ERROR:.*STILLRUNNING' && exit 1 || true
	@echo "run-primary-stops-while-service-is-running-test PASSED"
	

run-before-primary-test:
	@echo "\n\nrun-before-primary-test:"
	@echo "\tBefore the service starts, the -run commands should finish in order"
	@echo "################################################################################"
	@docker rm -f dockerfied-nginx 2>/dev/null || true
	docker run -d -it --name dockerfied-nginx dockerfied-nginx --verbose \
		--run bash -c 'sleep 3; echo "1 DONE"' -- \
		--run bash -c 'sleep 2; echo "2 DONE"' -- \
		--run bash -c 'echo "3 DONE"' -- \
		bash -c "sleep 4; echo '4 PRIMARY COMMAND DONE'"
	@sleep 15
	docker logs dockerfied-nginx 2>&1 | egrep -q '^1 DONE' 
	docker logs dockerfied-nginx 2>&1 | egrep -q '^2 DONE' 
	docker logs dockerfied-nginx 2>&1 | egrep -q '^3 DONE' 
	docker logs dockerfied-nginx 2>&1 | egrep -q '^4 PRIMARY COMMAND DONE' 
	@echo "run-before-primary-test PASSED"

run-primary-service-exits:
	@docker rm -f dockerfied-nginx-container 2>/dev/null || true
	@echo "\n\nrun-primary-service-exits:"
	@echo "\tVerify that when the primary service finishes, the container exits"
	@echo "################################################################################"
	docker run -it --name dockerfied-nginx-container dockerfied-nginx --verbose echo DONE 
	docker inspect --format '{{ .State.Status }}' dockerfied-nginx-container  | egrep -q exited
	@docker rm -f dockerfied-nginx-container 2>/dev/null || true
	@echo "run-primary-service-exits PASSED"


run-wait-test:
	@docker rm -f dockerfied-nginx-container 2>/dev/null || true
	@echo "\n\nrun-wait-test:"
	@echo "\tVerify that when the primary command runs after we wait for something"
	@echo "################################################################################"
	docker run -it --name dockerfied-nginx-container dockerfied-nginx --verbose \
		--wait http://amazon.com \
		echo DONE
	docker logs dockerfied-nginx-container 2>&1 | egrep -q '^DONE'
	@echo "run-wait-test PASSED"
	


