module github.com/example/webhook-engine

go 1.23.0

require (
    github.com/dgraph-io/badger/v4 v4.2.0
    github.com/sirupsen/logrus v1.9.3
    github.com/valyala/fasthttp v1.51.0
    github.com/spf13/pflag v1.0.5
    github.com/prometheus/client_golang v1.19.1
    go.opentelemetry.io/otel v1.28.0
    go.opentelemetry.io/otel/sdk v1.28.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
    google.golang.org/grpc v1.64.0
)

replace github.com/dgraph-io/badger/v4 => github.com/dgraph-io/badger/v4 v4.2.0
