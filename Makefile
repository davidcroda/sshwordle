.PHONE: build build_image

GITSHA=$(shell git rev-parse --short HEAD)

build:
	CGO_ENABLED=0 go build

build_image:
	docker build . -t localhost:5000/sshwordle:${GITSHA}
