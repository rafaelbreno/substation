# playground

Contains Substation apps deployed in the browser using WebAssembly (WASM). These provide similar functionality to the [Go Playground](https://go.dev/play/), but are run locally in the browser.

## wasm

This app runs Substation in the browser. Some features of Substation are not supported due to limitations in WASM. The WASM binary is built using these commands:

```sh
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o build/playground/playground.wasm cmd/playground/wasm/main.go && \
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" build/playground && \
cp substation_logo.png build/playground && \
gzip -9 -v -c build/playground/playground.wasm > build/playground/playground.wasm.gz && \
rm build/playground/playground.wasm
```

## server

This app starts a local server that serves the WASM binary. The server is started using this command:

```sh
go run cmd/playground/server/main.go
```
