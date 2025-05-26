.PHONY: run-lb

run-lb:
	go run cmd/loadbalancer/main.go

run-test:
	go test -v ./internal/*