.PHONY: all
all: wasm_exec.js main.wasm chart.js

GOROOT := $(shell go env GOROOT)

wasm_exec.js: $(GOROOT)/misc/wasm/wasm_exec.js
	cp $< $@

main.wasm: wasm/main.go
	$(MAKE) -C wasm

chart.js:
	printf "var chart = \"" > $@
	curl -Ls https://github.com/DataDog/helm-charts/releases/download/datadog-2.10.1/datadog-2.10.1.tgz | base64 -w0 >> $@
	printf "\"" >> $@
