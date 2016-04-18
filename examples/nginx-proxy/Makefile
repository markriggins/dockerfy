

nginx-proxy-image:
	docker build --tag nginx-proxy .


stop: 
	@docker rm -f nginx-proxy 2>/dev/null || true

proxy-amazon: nginx-proxy-image stop
	docker run -d -p 80:80 -e PROXY_PASS_URL="https://www.amazon.com/" --name nginx-proxy nginx-proxy
	open http://$(shell docker-machine ip $(shell docker-machine active))

proxy-wikipedia: nginx-proxy-image stop
	docker run -d -p 80:80 -e PROXY_PASS_URL="https://en.wikipedia.org/" --name nginx-proxy nginx-proxy
	open http://$(shell docker-machine ip $(shell docker-machine active))
