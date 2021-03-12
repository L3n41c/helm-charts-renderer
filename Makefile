.PHONY: all
all: wasm_exec.js main.wasm.gz chart.js

GOROOT := $(shell go env GOROOT)

wasm_exec.js: $(GOROOT)/misc/wasm/wasm_exec.js
	cp $< $@

.PHONY: wasm/main.wasm
wasm/main.wasm:
	$(MAKE) -C wasm

main.wasm.gz: wasm/main.wasm
	gzip --best --to-stdout $< > $@

chart.js:
	printf "var chart = \"" > $@
	curl -Ls https://github.com/DataDog/helm-charts/releases/download/datadog-2.10.1/datadog-2.10.1.tgz | base64 -w0 >> $@
	printf "\"" >> $@
