# Copyright 2019 The KubeCarrier Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c

export CGO_ENABLED:=0

ifdef CI
	# prow sets up GOPATH and we want to make sure it's in the PATH
	# https://github.com/kubernetes/test-infra/issues/9469
	# https://github.com/kubernetes/test-infra/blob/895df89b7e4238125063157842c191dac6f7e58f/prow/pod-utils/decorate/podspec.go#L474
	export PATH:=${PATH}:${GOPATH}/bin
endif

# run unittests
test:
	CGO_ENABLED=1 go test -race -v ./...
.PHONY: test

# lint project
lint:
	@hack/validate-directory-clean.sh
	pre-commit run -a
	golangci-lint run ./... --deadline=15m

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

install-git-hooks:
	pre-commit install
