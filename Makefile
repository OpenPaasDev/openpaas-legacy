.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:
	go clean -testcache
	go test ./... -race -covermode=atomic -coverprofile=coverage.out

.PHONY: coverage
coverage:
	go test -v -coverpkg=./... -coverprofile=profile.cov ./...
	go tool cover -func profile.cov

.PHONY: sync
sync:
	go run cmd/main.go sync --config.file=config.yaml

.PHONY: destroy
destroy:
	cd config/terraform
	terraform destroy -var="hcloud_token=$HETZNER_TOKEN"
	cd ../
	rm -rf config