.PHONY : dockerfy dist-clean dist release zombie-maker

TAG  := $(shell git describe --tags)
YTAG := $(shell git describe --tags | cut -d. -f1,2)
XTAG := $(shell git describe --tags | cut -d. -f1)
LDFLAGS:=-X main.buildVersion=$(TAG)
DOLLAR='$'

all: dockerfy nginx-with-dockerfy


prereqs:
	glide --no-color install -u -s -v


fmt:
	@echo gofmt:
	@test -z `glide novendor -x | sed '$$d' | xargs gofmt -l | tee /dev/stderr` && echo passed
	@echo


lint:
	@{ \
		if [ -z `which golint` ]; then \
			echo "golint not found in path. Available from: github.com/golang/lint/golint" ;\
			exit 1 ;\
		fi ;\
	}
	@echo golint:
	@glide novendor | xargs -n1 golint
	@echo


dockerfy: prereqs
	echo "Building dockerfy"
	go build -ldflags '$(LDFLAGS)'


debug: prereqs
	godebug run  $(ls *.go | egrep -v unix)


dist-clean:
	rm -rf dist
	rm -f dockerfy-linux-*.tar.gz


dist: dist-clean prereqs dist/linux/amd64/dockerfy nginx-with-dockerfy


# NOTE: this target is built by the above ^^ amd64 make inside a golang docker container
dist/linux/amd64/dockerfy: Makefile *.go
	mkdir -p dist/linux/amd64
	@# a native build allows user.Lookup to work.  Not sure why it doesn't if we cross-compile
	@# from OSX
	docker run --rm -it  \
	  --volume $$PWD/vendor:/go/src  \
	  --volume $$PWD:/go/src/dockerfy \
	  --workdir /go/src/dockerfy \
	  golang:1.7 go build -ldflags "$(LDFLAGS)" -o dist/linux/amd64/dockerfy


release: dist
	mkdir -p dist/release
	tar -czf dist/release/dockerfy-linux-amd64-$(TAG).tar.gz -C dist/linux/amd64 dockerfy
	@#tar -czf dist/release/dockerfy-linux-armel-$(TAG).tar.gz -C dist/linux/armel dockerfy
	@#tar -czf dist/release/dockerfy-linux-armhf-$(TAG).tar.gz -C dist/linux/armhf dockerfy


nginx-with-dockerfy:  dist/.mk.nginx-with-dockerfy


dist/.mk.nginx-with-dockerfy: Makefile dist/linux/amd64/dockerfy Dockerfile.nginx-with-dockerfy
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

test: fmt lint nginx-with-dockerfy
	cd test && make test

.PHONY: test
