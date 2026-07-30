module github.com/monstercameron/earlcameron

go 1.26.5

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/monstercameron/CashFlux v0.0.0-00010101000000-000000000000
	github.com/monstercameron/GoGRPCBridge v1.1.1
	github.com/monstercameron/GoWebComponents/v5 v5.0.1
	golang.org/x/crypto v0.54.0
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.54.0
)

// GWC tracks the local checkout (unreleased v5 — not on the module proxy, so the require version
// above is a placeholder the directory replace overrides). GoGRPCBridge is the published v1.1.1
// module. CashFlux still requires GWC /v4 v4.2.0, which resolves from the proxy as a separate major.
replace github.com/monstercameron/GoWebComponents/v5 => ../GoWebComponents

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-sqlite3 v0.35.0 // indirect
	github.com/ncruces/go-sqlite3-wasm/v3 v3.1.35302 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yuin/goldmark v1.7.13 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	modernc.org/libc v1.74.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/monstercameron/CashFlux => ../CashFlux
