CNTRCMD = podman

all: build

.PHONY: clean
clean:
	rm -f example embedded-binary
	$(CNTRCMD) image rm -f localhost/psi-example

.PHONY: build
build: embedded-binary example container

.PHONY: container
container:
	$(CNTRCMD) build -t localhost/psi-example:latest .

.PHONY: run
run:
	$(CNTRCMD) run -ti --rm localhost/psi-example:latest

.PHONY: run-interactive
run-interactive:
	$(CNTRCMD) run -ti --rm localhost/psi-example:latest interactive

embedded-binary:
	CGO_ENABLED=0 go build -trimpath -o embedded-binary ./embedded

example:
	CGO_ENABLED=0 go build -trimpath .
