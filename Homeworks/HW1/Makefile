IMAGE_NAME := githubstats
DOCKERFILE := docker/Dockerfile

# Cross-platform handling of paths for Docker (macOS + Windows Git Bash)
ifeq ($(OS),Windows_NT)
	PWD_CMD := pwd -W
	DOCKER_ENV_PREFIX := MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*'
else
	PWD_CMD := pwd
	DOCKER_ENV_PREFIX :=
endif

.PHONY: build run clean shell

## Build Docker image
build:
	docker build -t $(IMAGE_NAME) -f $(DOCKERFILE) .

## Run container and generate web/out.csv from usernames.txt
run: build
	$(DOCKER_ENV_PREFIX) docker run --rm \
		-v "$$($(PWD_CMD))":/data \
		-e GITHUB_TOKEN \
		$(IMAGE_NAME) /data/usernames.txt > web/out.csv
	@echo
	@echo "Done! Result written to web/out.csv"

## Open shell inside container (for debugging)
shell: build
	$(DOCKER_ENV_PREFIX) docker run --rm -it \
		-v "$$($(PWD_CMD))":/data \
		-e GITHUB_TOKEN \
		$(IMAGE_NAME) /bin/sh

## Clean generated files
clean:
	rm -f web/out.csv
