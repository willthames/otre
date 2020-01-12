VERSION = ${shell(git describe --long)}
image:
	docker build -t willthames/otre:${VERSION} .
push:
	docker push willthames/otre:${VERSION}
