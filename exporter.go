package apex

import (
	"context"
	"errors"
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
}

// Craetes a new App Insights Exporter with a service name
// Logger function is called for each diagnostic message
func NewExporter(
	instrumentationKey string,
	logger func(msg string) error,
) (*AppInsightsExporter, error) {
	client := appinsights.NewTelemetryClient(instrumentationKey)
	appinsights.NewDiagnosticsMessageListener(logger)
	return &AppInsightsExporter{
		client: client,
	}, nil
}

// Exports an array of Open Telemetry spans to Application Insights
func (exp *AppInsightsExporter) ExportSpans(
	ctx context.Context,
	spans []sdktrace.ReadOnlySpan,
) error {
	for _, e := range spans {
		exp.process(e)
	}
	return nil
}

// Exports an array of Open Telemetry spans to Application Insights
func (exp *AppInsightsExporter) Shutdown(
	ctx context.Context,
) error {
	select {
	case <-exp.client.Channel().Close(time.Minute):
		return nil
	case <-ctx.Done():
		return errors.New("context canceled")
	}
}

// Constructs the telemetry for an internal event and sends it to the client.
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

// Constructs the telemetry for a request and sends it to the client.
// Optional properties are examined for concrete fields and removed from the map.
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

// Constructs the telemetry for an event consumed and sends it to the client.
// Optional properties are examined for concrete fields and removed from the map.
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

// Constructs the telemetry for a dependency and sends it to the client.
// Optional properties are examined for concrete fields and removed from the map.
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

// Preprocesses the Otel span and dispatches it to app insights differently
// based on the span kind.
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
