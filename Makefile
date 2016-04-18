.PHONY : dockerfy dist-clean dist release zombie-maker

TAG:=`git describe --abbrev=0 --tags`
YTAG:=`git describe --abbrev=0 --tags | cut -d. -f1,2`
XTAG:=`git describe --abbrev=0 --tags | cut -d. -f1`
LDFLAGS:=-X main.buildVersion=$(TAG)
DOLLAR='$'

all: dockerfy nginx-with-dockerfy
	
dockerfy:
	echo "Building dockerfy"
	go install -ldflags "$(LDFLAGS)"


dist-clean:
	rm -rf dist
	rm -f dockerfy-linux-*.tar.gz

dist: dist-clean
	mkdir -p dist/linux/amd64 && GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/linux/amd64/dockerfy
	mkdir -p dist/linux/armel && GOOS=linux GOARCH=arm GOARM=5 go build -ldflags "$(LDFLAGS)" -o dist/linux/armel/dockerfy
	mkdir -p dist/linux/armhf && GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "$(LDFLAGS)" -o dist/linux/armhf/dockerfy

release: dist
	mkdir -p dist/release
	tar -czf dist/release/dockerfy-linux-amd64-$(TAG).tar.gz -C dist/linux/amd64 dockerfy 
	tar -czf dist/release/dockerfy-linux-armel-$(TAG).tar.gz -C dist/linux/armel dockerfy 
	tar -czf dist/release/dockerfy-linux-armhf-$(TAG).tar.gz -C dist/linux/armhf dockerfy 

nginx-with-dockerfy: dist
	docker build -t markriggins/nginx-with-dockerfy:$(TAG) --file Dockerfile.nginx-with-dockerfy .


float-tags: nginx-with-dockerfy
	# fail if we're not on a pure Z tag
	git describe --tags | egrep -q '^[0-9\.]+$$' 
	docker tag markriggins/nginx-with-dockerfy:$(TAG) markriggins/nginx-with-dockerfy:$(YTAG)
	docker tag markriggins/nginx-with-dockerfy:$(TAG) markriggins/nginx-with-dockerfy:$(XTAG)

push:
	docker push markriggins/nginx-with-dockerfy