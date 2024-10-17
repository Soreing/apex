package apex

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	trace "go.opentelemetry.io/otel/trace"
)

type AppInsightsExporter struct {
	client appinsights.TelemetryClient
	mtx    *sync.RWMutex
	closed bool
}

// NewExporter creates a new App Insights Exporter with an app insights
// telemetry client created from an instrumentation key. The exporter uses a
// logger function provided as a callback for logging events.
func NewExporter(
	instrumentationKey string,
	logger func(msg string) error,
) (*AppInsightsExporter, error) {
	client := appinsights.NewTelemetryClient(instrumentationKey)
	appinsights.NewDiagnosticsMessageListener(logger)
	return &AppInsightsExporter{
		client: client,
		mtx:    &sync.RWMutex{},
		closed: false,
	}, nil
}

// NewExporterFromConfig creates a new App Insights Exporter with an app
// insights telemetry client created from a telemetry configuration. The
// exporter uses a logger function provided as a callback for logging events.
func NewExporterFromConfig(
	cfg *appinsights.TelemetryConfiguration,
	logger func(msg string) error,
) (*AppInsightsExporter, error) {
	if cfg == nil {
		return nil, errors.New("configuration is nil")
	}

	client := appinsights.NewTelemetryClientFromConfig(cfg)
	appinsights.NewDiagnosticsMessageListener(logger)
	return &AppInsightsExporter{
		client: client,
		mtx:    &sync.RWMutex{},
		closed: false,
	}, nil
}

// ExportSpans processes and dispatches an array of Open Telemetry spans
// to Application Insights.
func (exp *AppInsightsExporter) ExportSpans(
	ctx context.Context,
	spans []sdktrace.ReadOnlySpan,
) error {
	exp.mtx.RLock()
	defer exp.mtx.RUnlock()

	if exp.closed {
		return errors.New("exporter closed")
	}

	for i := range spans {
		exp.process(spans[i])
	}
	return nil
}

// Shutdown closes the exporter and waits until the pending messages are sent
// with up to one minute grace period, or until the context is canceled.
// Grace period might change in the future to be optionable
func (exp *AppInsightsExporter) Shutdown(
	ctx context.Context,
) error {
	exp.mtx.Lock()
	defer exp.mtx.Unlock()
	exp.closed = true

	select {
	case <-exp.client.Channel().Close(time.Minute):
		return nil
	case <-ctx.Done():
		return errors.New("context canceled")
	}
}

// processInternal constructs a telemetry for an internal event and dispatches
// it to the application insights telemetry client.
//
// Application Insights specific fields are sourced from custom properties:
// Role = properties["service.name"]
func (exp *AppInsightsExporter) processInternal(
	sp sdktrace.ReadOnlySpan,
	properties map[string]string,
) {
	tele := appinsights.EventTelemetry{
		Name: sp.Name(),
		BaseTelemetry: appinsights.BaseTelemetry{
			Timestamp:  sp.StartTime(),
			Tags:       make(contracts.ContextTags),
			Properties: map[string]string{},
		},
		BaseTelemetryMeasurements: appinsights.BaseTelemetryMeasurements{
			Measurements: map[string]float64{},
		},
	}

	pid := sp.Parent().SpanID().String()
	if pid == "0000000000000000" {
		pid = sp.SpanContext().TraceID().String()
	}

	tele.Tags.Cloud().SetRole("unknown-service")
	if val, ok := properties[string(semconv.ServiceNameKey)]; ok {
		delete(properties, string(semconv.ServiceNameKey))
		tele.Tags.Cloud().SetRole(val)
	}
	tele.BaseTelemetry.Properties = properties

	tele.Tags.Operation().SetId(sp.SpanContext().TraceID().String())
	tele.Tags.Operation().SetParentId(pid)
	tele.Tags.Operation().SetName(sp.Name())

	exp.client.Track(&tele)
}

// processRequest constructs the telemetry for an incoming http request
// and and dispatches it to the application insights telemetry client.
//
// Application Insights specific fields are sourced from custom properties:
// Role = properties["service.name"]
// Url = properties["url"]
// ResponseCode = properties["responseCode"]
func (exp *AppInsightsExporter) processRequest(
	sp sdktrace.ReadOnlySpan,
	success bool,
	properties map[string]string,
) {
	tele := appinsights.RequestTelemetry{
		Name:         sp.Name(),
		Url:          "",
		Id:           sp.SpanContext().SpanID().String(),
		Duration:     sp.EndTime().Sub(sp.StartTime()),
		ResponseCode: "0",
		Success:      success,
		BaseTelemetry: appinsights.BaseTelemetry{
			Timestamp:  sp.StartTime(),
			Tags:       make(contracts.ContextTags),
			Properties: map[string]string{},
		},
		BaseTelemetryMeasurements: appinsights.BaseTelemetryMeasurements{
			Measurements: map[string]float64{},
		},
	}
	tele.Tags.Cloud().SetRole("unknown-service")
	if val, ok := properties[string(semconv.ServiceNameKey)]; ok {
		delete(properties, string(semconv.ServiceNameKey))
		tele.Tags.Cloud().SetRole(val)
	}
	if val, ok := properties["url"]; ok {
		delete(properties, "url")
		tele.Url = val
	}
	if val, ok := properties["responseCode"]; ok {
		delete(properties, "responseCode")
		tele.ResponseCode = val
	}
	tele.BaseTelemetry.Properties = properties

	pid := sp.Parent().SpanID().String()
	if pid == "0000000000000000" {
		pid = sp.SpanContext().TraceID().String()
	}

	tele.Tags.Operation().SetId(sp.SpanContext().TraceID().String())
	tele.Tags.Operation().SetParentId(pid)
	tele.Tags.Operation().SetName(sp.Name())

	exp.client.Track(&tele)
}

// processEvent constructs the telemetry for an incoming event to be handled
// and and dispatches it to the application insights telemetry client.
//
// Application Insights specific fields are sourced from custom properties:
// Role = properties["service.name"]
// Url = properties["key"]
// ResponseCode = properties["responseCode"]
func (exp *AppInsightsExporter) processEvent(
	sp sdktrace.ReadOnlySpan,
	success bool,
	properties map[string]string,
) {
	tele := appinsights.RequestTelemetry{
		Name:         sp.Name(),
		Url:          "",
		Id:           sp.SpanContext().SpanID().String(),
		Duration:     sp.EndTime().Sub(sp.StartTime()),
		ResponseCode: "0",
		Success:      success,
		BaseTelemetry: appinsights.BaseTelemetry{
			Timestamp:  sp.StartTime(),
			Tags:       make(contracts.ContextTags),
			Properties: map[string]string{},
		},
		BaseTelemetryMeasurements: appinsights.BaseTelemetryMeasurements{
			Measurements: map[string]float64{},
		},
	}
	tele.Tags.Cloud().SetRole("unknown-service")
	if val, ok := properties[string(semconv.ServiceNameKey)]; ok {
		delete(properties, string(semconv.ServiceNameKey))
		tele.Tags.Cloud().SetRole(val)
	}
	if val, ok := properties["key"]; ok {
		delete(properties, "key")
		tele.Url = val
	}
	if val, ok := properties["responseCode"]; ok {
		delete(properties, "responseCode")
		tele.ResponseCode = val
	}
	tele.BaseTelemetry.Properties = properties

	pid := sp.Parent().SpanID().String()
	if pid == "0000000000000000" {
		pid = sp.SpanContext().TraceID().String()
	}

	tele.Tags.Operation().SetId(sp.SpanContext().TraceID().String())
	tele.Tags.Operation().SetParentId(pid)
	tele.Tags.Operation().SetName(sp.Name())

	exp.client.Track(&tele)
}

// processDependency constructs the telemetry for an outgoing dependency
// and and dispatches it to the application insights telemetry client.
//
// Application Insights specific fields are sourced from custom properties:
// Role = properties["source"]
// Type = properties["type"]
// Target = properties["service.name"]
func (exp *AppInsightsExporter) processDependency(
	sp sdktrace.ReadOnlySpan,
	success bool,
	properties map[string]string,
) {
	tele := appinsights.RemoteDependencyTelemetry{
		Name:     sp.Name(),
		Id:       sp.SpanContext().SpanID().String(),
		Type:     "",
		Target:   "",
		Duration: sp.EndTime().Sub(sp.StartTime()),
		Success:  success,
		BaseTelemetry: appinsights.BaseTelemetry{
			Timestamp:  sp.StartTime(),
			Tags:       make(contracts.ContextTags),
			Properties: map[string]string{},
		},
		BaseTelemetryMeasurements: appinsights.BaseTelemetryMeasurements{
			Measurements: map[string]float64{},
		},
	}
	tele.Tags.Cloud().SetRole("unknown-service")
	if val, ok := properties["source"]; ok {
		delete(properties, "source")
		tele.Tags.Cloud().SetRole(val)
	}
	if val, ok := properties["type"]; ok {
		delete(properties, "type")
		tele.Type = val
	}
	tele.Target = "unknown-target"
	if val, ok := properties[string(semconv.ServiceNameKey)]; ok {
		delete(properties, string(semconv.ServiceNameKey))
		tele.Target = val
	}
	tele.BaseTelemetry.Properties = properties

	pid := sp.Parent().SpanID().String()
	if pid == "0000000000000000" {
		pid = sp.SpanContext().TraceID().String()
	}

	tele.Tags.Operation().SetId(sp.SpanContext().TraceID().String())
	tele.Tags.Operation().SetParentId(pid)
	tele.Tags.Operation().SetName(sp.Name())

	exp.client.Track(&tele)
}

// process routes the span to different processing functions based on the
// span's kind to be processed appropriately
func (exp *AppInsightsExporter) process(sp sdktrace.ReadOnlySpan) {
	success := true
	if sp.Status().Code != codes.Ok {
		success = false
	}

	props := map[string]string{}

	rattr := sp.Resource().Attributes()
	for _, e := range rattr {
		props[string(e.Key)] = e.Value.AsString()
	}
	attr := sp.Attributes()
	for _, e := range attr {
		props[string(e.Key)] = e.Value.AsString()
	}

	switch sp.SpanKind() {
	case trace.SpanKindUnspecified:
		exp.processInternal(sp, props)
	case trace.SpanKindInternal:
		exp.processInternal(sp, props)
	case trace.SpanKindServer:
		exp.processRequest(sp, success, props)
	case trace.SpanKindClient:
		exp.processDependency(sp, success, props)
	case trace.SpanKindProducer:
		exp.processDependency(sp, success, props)
	case trace.SpanKindConsumer:
		exp.processEvent(sp, success, props)
	}
}
