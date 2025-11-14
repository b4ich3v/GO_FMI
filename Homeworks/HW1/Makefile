IMAGE_NAME := githubstats
DOCKERFILE := docker/Dockerfile

.PHONY: build run clean shell

## Build Docker image
build:
	docker build -t $(IMAGE_NAME) -f $(DOCKERFILE) .

## Run container and generate web/out.csv from usernames.txt
run: build
	docker run --rm \
		-v "$$(pwd)":/data \
		-e GITHUB_TOKEN \
		$(IMAGE_NAME) /data/usernames.txt > web/out.csv
	@echo
	@echo "Done! Result written to web/out.csv"

## Open shell inside container (for debugging)
shell: build
	docker run --rm -it \
		-v "$$(pwd)":/data \
		-e GITHUB_TOKEN \
		$(IMAGE_NAME) /bin/sh

## Clean generated files
clean:
	rm -f web/out.csv
