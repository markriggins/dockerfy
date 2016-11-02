.PHONY : dockerfy dist-clean dist release zombie-maker

TAG  := $(shell git describe --tags --match '[0-9]*\.[0-9]*')
YTAG := $(shell echo $(TAG) | cut -d. -f1,2)
XTAG := $(shell echo $(TAG) | cut -d. -f1)
LDFLAGS:=-X main.buildVersion=$(TAG)
DOLLAR='$'
export SHELL = /bin/bash

all: dockerfy nginx-with-dockerfy


prereqs: .mk.glide


.mk.glide: glide.yaml
	glide --no-color install -u -s -v
	touch .mk.glide


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


is-open-source-clean:
	@{ \
		if glide -q list 2>/dev/null | egrep -iq 'github.com/SocialCodeInc'; then \
			echo "Dockerfy is OPEN SOURCE -- no dependencies on SocialCodeInc sources are allowed"; \
		else \
			echo "Dockerfy is clean for OPEN SOURCE"; \
		fi; \
	}


dockerfy: prereqs *.go
	echo "Building dockerfy"
	go build -ldflags '$(LDFLAGS)'


install: prereqs *.go
	echo "Building dockerfy"
	go install -ldflags '$(LDFLAGS)'

debug: prereqs
	godebug run  $(ls *.go | egrep -v unix)


dist-clean:
	rm -rf dist
	rm -f dockerfy-linux-*.tar.gz


dist: dist-clean dist/linux/amd64/dockerfy nginx-with-dockerfy


# NOTE: this target is built by the above ^^ amd64 make inside a golang docker container
dist/linux/amd64/dockerfy: prereqs Makefile *.go
	mkdir -p dist/linux/amd64
	@# a native build allows user.Lookup to work.  Not sure why it doesn't if we cross-compile
	@# from OSX
	docker run --rm  \
	  --volume $$PWD/vendor:/go/src  \
	  --volume $$PWD:/go/src/dockerfy \
	  --workdir /go/src/dockerfy \
	  golang:1.7 go build -ldflags "$(LDFLAGS)" -o dist/linux/amd64/dockerfy


is-clean-z-release:
	echo "TAG=$(TAG)"
	@{ \
		if [[ "$(TAG)" =~ .*\-[0-9]+\-g[0-9a-f]+$$ ]]; then \
		    last_tag=$$(git describe --tags --abbrev=0); \
	        echo "ERROR: there have been commits on this branch after the $$last_tag tag was created";  \
	        echo '       please create a fresh GIT Z tag `git tag -a X.Y.Z -m "description ..."`'; \
	        echo '       or build at an existing TAG instead of on a branch'; \
	        false; \
		elif ! [[ "$(TAG)" =~ ^[0-9]+(\.[0-9]+)+.* ]]; then \
	    	echo "$(TAG) is not a SEMVER Z tag"; \
	    	false; \
		fi; \
	}


release: is-clean-z-release dist
	mkdir -p dist/release
	tar -czf dist/release/dockerfy-linux-amd64-$(TAG).tar.gz -C dist/linux/amd64 dockerfy


publish: release
	hub release create -a dist/release/dockerfy-linux-amd64-$(TAG).tar.gz -m'$(TAG)' $(TAG)


nginx-with-dockerfy:  dist/.mk.nginx-with-dockerfy


dist/.mk.nginx-with-dockerfy: Makefile dist/linux/amd64/dockerfy Dockerfile.nginx-with-dockerfy
	docker build -t socialcode/nginx-with-dockerfy:$(TAG) --file Dockerfile.nginx-with-dockerfy .
	docker tag socialcode/nginx-with-dockerfy:$(TAG) nginx-with-dockerfy
	touch dist/.mk.nginx-with-dockerfy


float-tags: is-clean-z-release  nginx-with-dockerfy
	docker tag socialcode/nginx-with-dockerfy:$(TAG) socialcode/nginx-with-dockerfy:$(YTAG)
	docker tag socialcode/nginx-with-dockerfy:$(TAG) socialcode/nginx-with-dockerfy:$(XTAG)


push: float-tags
	docker images | grep nginx-with-dockerfy
	# pushing the entire repository will push all tagged images
	docker push socialcode/nginx-with-dockerfy


test: fmt lint is-open-source-clean nginx-with-dockerfy
	cd test && make test


test-and-log: fmt lint nginx-with-dockerfy
	cd test && make test-and-log


.PHONY: test
