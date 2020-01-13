VERSION := $(shell git describe --long)
image:
	docker build -t willthames/otre:${VERSION} .
push: image
	docker push willthames/otre:${VERSION}
