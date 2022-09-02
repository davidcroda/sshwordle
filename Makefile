.PHONE: build build_image

GITSHA=$(shell git rev-parse --short HEAD)

build:
	docker-compose build

dev:
	docker-compose up

build_image:
	docker build . -t localhost:5000/sshwordle:${GITSHA}
