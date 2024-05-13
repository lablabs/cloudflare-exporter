.PHONY: build
build: lint
	CGO_ENABLED=0 go build --ldflags '-w -s -extldflags "-static"' -o cloudflare_exporter .
lint:
	golangci-lint run
clean:
	rm cloudflare_exporter venom*.log basic_tests.* pprof_cpu*
test:
	./run_e2e.sh
