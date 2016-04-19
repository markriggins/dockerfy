# proxy_pass {{ .Env.ROOT_SCHEME }}://{{ .Env.HOSTNAME }}:{{ .Env.ROOT_PORT }}{{ .Env.ROOT_PATH }}/;
#proxy_redirect {{ .Env.APP_1_SCHEME }}://{{ .Env.HOSTNAME }}:{{ .Env.APP_1_PORT }}{{ .Env.APP_1_PATH }} $host;}

DOCKER_IP := $(shell docker-machine ip $(shell docker-machine active))
performance-proxy-image:
	docker build --tag performance-proxy .


stop: 
	@docker rm -f performance-proxy 2>/dev/null || true
	@docker rm -f todowranger 2>/dev/null || true

proxy-performance: performance-proxy-image stop
	docker run -d -p 8080:80 -e ROOT_SCHEME="http" -e ROOT_PORT="8000" -e ROOT_PATH="" -e APP_1_SCHEME="http" -e APP_1_PORT="4200" -e APP_1_PATH="/performance/app-1" -e APP_1_RULE="/performance/app-1"  --name performance-proxy performance-proxy
	open http://$(shell docker-machine ip $(shell docker-machine active))

