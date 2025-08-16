package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Options struct {
	ServiceName  string
	SampleRatio  float64
	OTLPEndpoint string // host:port or http://host:port
}

func Setup(opts Options) (func(context.Context) error, error) {
	res, _ := sdkresource.New(
		context.Background(),
		sdkresource.WithAttributes(semconv.ServiceName(opts.ServiceName)),
	)

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(opts.SampleRatio))

	var tp *sdktrace.TracerProvider
	if opts.OTLPEndpoint != "" {
		addr := trimScheme(opts.OTLPEndpoint)
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil { return func(context.Context) error { return nil }, err }
		exp, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
		if err != nil { return func(context.Context) error { return nil }, err }
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sampler),
			sdktrace.WithBatcher(exp, sdktrace.WithMaxExportBatchSize(256), sdktrace.WithBatchTimeout(500*time.Millisecond)),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.TraceContext{})
		return tp.Shutdown, nil
	}
	// no exporter
	tp = sdktrace.NewTracerProvider(sdktrace.WithSampler(sampler), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp.Shutdown, nil
}

func trimScheme(s string) string {
	if len(s) > 7 && s[:7] == "http://" { return s[7:] }
	if len(s) > 8 && s[:8] == "https://" { return s[8:] }
	return s
}
