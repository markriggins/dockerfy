.PHONY : dockerfy dist-clean dist release zombie-maker

TAG:=`git describe --tags`
YTAG:=`git describe --tags | cut -d. -f1,2`
XTAG:=`git describe --tags | cut -d. -f1`
LDFLAGS:=-X main.buildVersion=$(TAG)
DOLLAR='$'

all: dockerfy nginx-with-dockerfy

dockerfy: deps
	echo "Building dockerfy"
	go install -ldflags "$(LDFLAGS)"

debug: deps
	godebug run  $(ls *.go | egrep -v unix)

deps:
	go get github.com/hpcloud/tail
	go get golang.org/x/net/context
	go get golang.org/x/sys/unix


dist-clean:
	rm -rf dist
	rm -f dockerfy-linux-*.tar.gz


dist: dist-clean deps build-dist-dockerfy-in-container nginx-with-dockerfy

build-dist-dockerfy-in-container:
	mkdir -p dist/linux/amd64
	@# a native build allows user.Lookup to work.  Not sure why it doesn't if we cross-compile
	@# from OSX
	docker run --rm -it -e GOPATH=/tmp/go \
	  --volume $$PWD:/src/dockerfy  \
	  --workdir /src/dockerfy \
	  golang:1.6 make dist/linux/amd64/dockerfy

# NOTE: this target is built by the above ^^ amd64 make inside an golang:1.6 docker container
dist/linux/amd64/dockerfy: deps Makefile
	go build -ldflags "$(LDFLAGS)" -o dist/linux/amd64/dockerfy


release: dist
	mkdir -p dist/release
	tar -czf dist/release/dockerfy-linux-amd64-$(TAG).tar.gz -C dist/linux/amd64 dockerfy
	@#tar -czf dist/release/dockerfy-linux-armel-$(TAG).tar.gz -C dist/linux/armel dockerfy
	@#tar -czf dist/release/dockerfy-linux-armhf-$(TAG).tar.gz -C dist/linux/armhf dockerfy

nginx-with-dockerfy: dist/.mk.nginx-with-dockerfy

dist/.mk.nginx-with-dockerfy: Makefile
	docker build -t markriggins/nginx-with-dockerfy:$(TAG) --file Dockerfile.nginx-with-dockerfy .
	docker tag markriggins/nginx-with-dockerfy:$(TAG) nginx-with-dockerfy
	touch dist/.mk.nginx-with-dockerfy


float-tags: nginx-with-dockerfy
	# fail if we're not on a pure Z tag
	git describe --tags | egrep -q '^[0-9\.]+$$'
	docker tag markriggins/nginx-with-dockerfy:$(TAG) markriggins/nginx-with-dockerfy:$(YTAG)
	docker tag markriggins/nginx-with-dockerfy:$(TAG) markriggins/nginx-with-dockerfy:$(XTAG)

push:
	docker push markriggins/nginx-with-dockerfy