
DOCKER_IP := $(shell docker-machine ip $(shell docker-machine active))
nginx-proxy-image:
	docker build --tag nginx-proxy .


stop: 
	@docker rm -f nginx-proxy 2>/dev/null || true
	@docker rm -f todowranger 2>/dev/null || true


#
#  This example shows how to pass an environment variable $PROXY_PASS_URL into the nginx.conf file by 
#  using a template.   It can proxy amazon.com or any other website that works with nginx reverse proxy
#
proxy-amazon: nginx-proxy-image stop
	docker run -d -p 80:80 -e PROXY_PASS_URL="https://www.amazon.com/" --name nginx-proxy nginx-proxy
	open http://$(shell docker-machine ip $(shell docker-machine active))


#
#  This example show how to proxy back to another container running on the same host
#
proxy-todowranger: nginx-proxy-image stop
	docker run -d -p 8001:80 --name todowrangler markriggins/todowrangler
	docker run -d -p 80:80 -e PROXY_PASS_URL="http://$(DOCKER_IP):8001" --name nginx-proxy nginx-proxy
	open http://$(DOCKER_IP)