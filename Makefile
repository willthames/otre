image:
	docker build -t willthames/otre:${VERSION} .
push:
	docker push willthames/otre:${VERSION}
