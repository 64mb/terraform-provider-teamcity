export GO111MODULE=on

GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
BUILDER_IMAGE=64mb/terraform-provider-teamcity-builder
VERSION=0.0.1

default: test

build:
	GO111MODULE=on go build -o ./bin/terraform-provider-teamcity_v${VERSION} . && chmod +x ./bin/terraform-provider-teamcity_v${VERSION}

build-linux-amd64:
	mkdir -p ./bin/linux_amd64 && GOOS=linux GO111MODULE=on go build -o ./bin/linux_amd64/terraform-provider-teamcity_v${VERSION} . && chmod +x ./bin/linux_amd64/terraform-provider-teamcity_v${VERSION}

build-darwin-amd64:
	mkdir -p ./bin/darwin_amd64 && GOOS=darwin GOARCH=amd64 GO111MODULE=on go build -o ./bin/darwin_amd64/terraform-provider-teamcity_v${VERSION} . && chmod +x ./bin/darwin_amd64/terraform-provider-teamcity_v${VERSION}

build-darwin-arm64:
	mkdir -p ./bin/darwin_arm64 && GOOS=darwin GOARCH=arm64 GO111MODULE=on go build -o ./bin/darwin_arm64/terraform-provider-teamcity_v${VERSION} . && chmod +x ./bin/darwin_arm64/terraform-provider-teamcity_v${VERSION}

build-windows-amd64:
	mkdir -p ./bin/windows_amd64 && GOOS=windows GOARCH=arm64 GO111MODULE=on go build -o ./bin/windows_amd64/terraform-provider-teamcity_v${VERSION} . && chmod +x ./bin/windows_amd64/terraform-provider-teamcity_v${VERSION}

build-all: build-linux-amd64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

install: build
	mkdir -p ~/.terraform.d/plugins/terraform.local/64mb/teamcity/${VERSION}/linux_amd64 && cp ./bin/terraform-provider-teamcity_v${VERSION} ~/.terraform.d/plugins/terraform.local/64mb/teamcity/${VERSION}/linux_amd64

clean:
	rm -rf ./bin

builder-action:
	docker run -e GITHUB_WORKSPACE='/github/workspace' -e GITHUB_REPOSITORY='terraform-provider-teamcity' -e GITHUB_REF='v0.0.1-alpha' --name terraform-provider-teamcity-builder $(BUILDER_IMAGE):latest

builder-image:
	docker build .github/builder --tag $(BUILDER_IMAGE)

clean_samples:
	find ./examples -name '*.tfstate' -delete
	find ./examples -name ".terraform" -type d -exec rm -rf "{}" \;

fmt_samples:
	terraform fmt -recursive examples/
