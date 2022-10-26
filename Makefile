.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:
	go clean -testcache
	go test ./... -race -covermode=atomic -coverprofile=coverage.out

.PHONY: sync
sync:
	go run main.go sync --config.file=config.yaml

.PHONY: destroy
destroy:
	cd config
	terraform destroy -var="hcloud_token=$HETZNER_TOKEN"
	cd ../
	rm -rf config