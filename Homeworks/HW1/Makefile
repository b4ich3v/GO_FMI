IMAGE_NAME := githubstats

.PHONY: build run clean shell

build:
	docker build -t $(IMAGE_NAME) .

run: build
	docker run --rm \
		-v "$$(pwd)":/data \
		-e GITHUB_TOKEN \
		$(IMAGE_NAME) /data/usernames.txt > out.csv
	@echo
	@echo "Done! The result is written in out.csv file"

shell: build
	docker run --rm -it \
		-v "$$(pwd)":/data \
		-e GITHUB_TOKEN \
		$(IMAGE_NAME) /bin/sh

clean:
	rm -f out.csv
