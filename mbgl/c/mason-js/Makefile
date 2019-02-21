SHELL = /bin/bash

build:
	docker build \
		--tag mason-js:latest .

bash:
	docker run \
		-it mason-js:latest bash

run: build
	docker run \
		-it mason-js:latest

test: build
	docker run \
		--tty \
		--rm \
		mason-js:latest \
		npm test
