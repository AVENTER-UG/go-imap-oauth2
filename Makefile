#Dockerfile vars
#vars
IMAGENAME=go-imap-oauth2
REPO=avhost
TAG=v0.1.0
BRANCH=${TAG}
BRANCHSHORT=$(shell echo ${BRANCH} | awk -F. '{ print $$1"."$$2 }')
BUILDDATE=$(shell date -u +%Y%m%d)
IMAGEFULLNAME=${REPO}/${IMAGENAME}
LASTCOMMIT=$(shell git log -1 --pretty=short | tail -n 1 | tr -d " " | tr -d "UPDATE:")


.PHONY: help build all docs

.DEFAULT_GOAL := all

build:
	@echo ">>>> Build Docker: latest"
	@docker build --build-arg TAG=${TAG} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:latest .

build-bin:
	@echo ">>>> Build binary"
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.BuildVersion=${BUILDDATE} -X main.GitVersion=${TAG} -extldflags \"-static\"" .

push:
	@echo ">>>> Publish docker image: " ${BRANCH} ${BRANCHSHORT}
	-docker buildx create --use --name buildkit
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${BRANCH} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:${BRANCH} .
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${BRANCH} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:${BRANCHSHORT} .
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${BRANCH} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:latest .


update-gomod:
	go get -u
	go mod tidy

plugin: 
	@echo ">>> Build plugins"
	cd plugins; $(MAKE)

sboom:
	syft dir:. > sbom.txt
	syft dir:. -o json > sbom.json

seccheck:
	grype --add-cpes-if-none .

imagecheck:
	grype --add-cpes-if-none ${IMAGEFULLNAME}:latest > cve-report.md

go-fmt:
	@gofmt -w .

check: go-fmt sboom seccheck
all: check build imagecheck
