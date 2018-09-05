REPO=warden
TAG ?= latest
BUILD_TAG ?= latest
DOCKER_REGISTRY=095427547185.dkr.ecr.us-west-2.amazonaws.com
IMAGE_TAG=$(DOCKER_REGISTRY)/skuid/$(REPO):$(BUILD_TAG)
SHA = $(shell git rev-parse --short HEAD)

.PHONY: clean

all: build

test:
	@go test -cover ./...

start:
	@docker-compose up -d postgres redis
	@./warden serve \
		--pprof=true \
		--tls_enabled=true \
		--database_on=true \
		--local_aws_conf_enabled=true

profile:
	@go tool pprof https://localhost:$(WARDEN_PORT)/debug/pprof/profile

migrate:
	@./warden migrate up \
		--database_on=true \
		--local_aws_conf_enabled=true

build:
	docker run --rm \
		-v $$(pwd):/go/src/github.com/skuid/$(REPO) \
		-w /go/src/github.com/skuid/$(REPO) \
		golang:1.9.1 \
			go build -v -a -tags netgo -installsuffix netgo -ldflags '-w'
	docker build -t skuid/$(REPO):$(TAG) .

ci-artifacts:
	mkdir -p artifacts
	echo "$(IMAGE_TAG)" > artifacts/docker-image.txt
	echo "$(SHA)" > artifacts/revision.txt

ci-build:
	docker build -f Dockerfile.build -t skuid/$(REPO):$(TAG)-build .
	docker create --name appLoader-$(TAG) skuid/$(REPO):$(TAG)-build
	docker cp appLoader-$(TAG):/go/bin/$(REPO) .
	docker rm appLoader-$(TAG)
	docker build -t skuid/$(REPO):$(TAG) .

ci-upload: ci-build ci-artifacts
	$$(aws ecr get-login)
	docker tag skuid/$(REPO):$(TAG) $(IMAGE_TAG)
	docker push $(IMAGE_TAG)

test-us: build
	ACCOUNT=095427547185
	ENVIRON=test
	REGION=us-west-2
	$$(aws ecr get-login --region ${REGION} --profile ${ENVIRON})
	docker tag skuid/$(REPO):$(TAG) ${ACCOUNT}.dkr.ecr.${REGION}.amazonaws.com/skuid/$(REPO):$(TAG)
	docker push ${ACCOUNT}.dkr.ecr.${REGION}.amazonaws.com/skuid/$(REPO):$(TAG)

prod-us: build
	ACCOUNT=775670380454
	ENVIRON=prod
	REGION=us-west-2
	$$(aws ecr get-login --region ${REGION} --profile ${ENVIRON})
	docker tag skuid/$(REPO):$(TAG) ${ACCOUNT}.dkr.ecr.${REGION}.amazonaws.com/skuid/$(REPO):$(TAG)
	docker push ${ACCOUNT}.dkr.ecr.${REGION}.amazonaws.com/skuid/$(REPO):$(TAG)

prod-eu: build
	ACCOUNT=775670380454
	ENVIRON=prod-eu
	REGION=eu-central-1
	$$(aws ecr get-login --region ${REGION} --profile ${ENVIRON})
	docker tag skuid/$(REPO):$(TAG) ${ACCOUNT}.dkr.ecr.${REGION}.amazonaws.com/skuid/$(REPO):$(TAG)
	docker push ${ACCOUNT}.dkr.ecr.${REGION}.amazonaws.com/skuid/$(REPO):$(TAG)

clean:
	rm ./$(REPO)
