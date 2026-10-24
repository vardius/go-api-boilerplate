package obs

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Setup installs the global OTel tracer and meter providers and the W3C
// TraceContext and Baggage propagators.
//
// Traces are exported over OTLP/HTTP when OTEL_EXPORTER_OTLP_ENDPOINT or
// OTEL_EXPORTER_OTLP_TRACES_ENDPOINT is set, and metrics when
// OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_METRICS_ENDPOINT is set;
// the exporters read the rest of the standard OTEL_* variables themselves.
// A signal with no endpoint gets a no-op provider. Call shutdown before the
// process exits to flush what is buffered.
func Setup(ctx context.Context, service, version string) (shutdown func(context.Context) error, err error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	traces := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
	metrics := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") != ""

	var tp trace.TracerProvider = tracenoop.NewTracerProvider()
	var mp metric.MeterProvider = metricnoop.NewMeterProvider()
	var shutdowns []func(context.Context) error
	shutdownAll := func(ctx context.Context) error {
		var errs []error
		for _, f := range shutdowns {
			errs = append(errs, f(ctx))
		}
		return errors.Join(errs...)
	}

	if traces || metrics {
		// Variables set in the environment (OTEL_SERVICE_NAME,
		// OTEL_RESOURCE_ATTRIBUTES) override the service's own name.
		res, err := resource.New(ctx,
			resource.WithAttributes(semconv.ServiceName(service), semconv.ServiceVersion(version)),
			resource.WithFromEnv(),
			resource.WithTelemetrySDK(),
		)
		if err != nil {
			return nil, fmt.Errorf("obs: describe the resource: %w", err)
		}
		if traces {
			exp, err := otlptracehttp.New(ctx)
			if err != nil {
				return nil, fmt.Errorf("obs: trace exporter: %w", err)
			}
			sdk := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
			tp = sdk
			shutdowns = append(shutdowns, sdk.Shutdown)
		}
		if metrics {
			exp, err := otlpmetrichttp.New(ctx)
			if err != nil {
				return nil, errors.Join(fmt.Errorf("obs: metric exporter: %w", err), shutdownAll(ctx))
			}
			sdk := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)), sdkmetric.WithResource(res))
			mp = sdk
			shutdowns = append(shutdowns, sdk.Shutdown)
		}
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	return shutdownAll, nil
}
