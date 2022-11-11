package apex

import (
	"context"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	trace "go.opentelemetry.io/otel/trace"
)

type AppInsightsExporter struct {
	client   appinsights.TelemetryClient
	servName string
}

// Craetes a new App Insights Exporter with a service name
// Logger function is called for each diagnostic message
func NewExporter(
	instrumentationKey string,
	serviceName string,
	logger func(msg string) error,
) (*AppInsightsExporter, error) {
	client := appinsights.NewTelemetryClient(instrumentationKey)
	appinsights.NewDiagnosticsMessageListener(logger)
	return &AppInsightsExporter{
		client:   client,
		servName: serviceName,
	}, nil
}

// Exports an array of Open Telemetry spans to Application Insights
func (exp *AppInsightsExporter) ExportSpans(
	ctx context.Context,
	spans []sdktrace.ReadOnlySpan,
) error {
	for _, e := range spans {
		exp.Process(e)
	}
	return nil
}

// Exports an array of Open Telemetry spans to Application Insights
func (exp *AppInsightsExporter) MarshalLog() interface{} {
	return struct {
		Type string
		URL  string
	}{
		Type: "appInsights",
		URL:  "",
	}
}

// Exports an array of Open Telemetry spans to Application Insights
func (exp *AppInsightsExporter) Shutdown(
	ctx context.Context,
) error {
	select {
	case <-exp.client.Channel().Close(10 * time.Second):
	case <-time.After(30 * time.Second):
	}
	return nil
}

// Constructs the telemetry for an internal event and sends it to the client.
func (exp *AppInsightsExporter) ProcessInternal(
	sp sdktrace.ReadOnlySpan,
	properties map[string]string,
) {
	tele := appinsights.EventTelemetry{
		Name: sp.Name(),
		BaseTelemetry: appinsights.BaseTelemetry{
			Timestamp:  sp.StartTime(),
			Tags:       make(contracts.ContextTags),
			Properties: properties,
		},
		BaseTelemetryMeasurements: appinsights.BaseTelemetryMeasurements{
			Measurements: map[string]float64{},
		},
	}

	pid := sp.Parent().SpanID().String()
	if pid == "0000000000000000" {
		pid = sp.SpanContext().TraceID().String()
	}

	tele.Tags.Cloud().SetRole(exp.servName)
	tele.Tags.Operation().SetId(sp.SpanContext().TraceID().String())
	tele.Tags.Operation().SetParentId(pid)
	tele.Tags.Operation().SetName(sp.Name())

	exp.client.Track(&tele)
}

// Constructs the telemetry for a request and sends it to the client.
// Optional properties are examined for concrete fields and removed from the map.
func (exp *AppInsightsExporter) ProcessRequest(
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

	tele.Tags.Cloud().SetRole(exp.servName)
	tele.Tags.Operation().SetId(sp.SpanContext().TraceID().String())
	tele.Tags.Operation().SetParentId(pid)
	tele.Tags.Operation().SetName(sp.Name())

	exp.client.Track(&tele)
}

// Constructs the telemetry for an event consumed and sends it to the client.
// Optional properties are examined for concrete fields and removed from the map.
func (exp *AppInsightsExporter) ProcessEvent(
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

	tele.Tags.Cloud().SetRole(exp.servName)
	tele.Tags.Operation().SetId(sp.SpanContext().TraceID().String())
	tele.Tags.Operation().SetParentId(pid)
	tele.Tags.Operation().SetName(sp.Name())

	exp.client.Track(&tele)
}

// Constructs the telemetry for a dependency and sends it to the client.
// Optional properties are examined for concrete fields and removed from the map.
func (exp *AppInsightsExporter) ProcessDependency(
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
	if val, ok := properties["type"]; ok {
		delete(properties, "type")
		tele.Type = val
	}
	if val, ok := properties["target"]; ok {
		delete(properties, "target")
		tele.Target = val
	}
	tele.BaseTelemetry.Properties = properties

	pid := sp.Parent().SpanID().String()
	if pid == "0000000000000000" {
		pid = sp.SpanContext().TraceID().String()
	}

	tele.Tags.Cloud().SetRole(exp.servName)
	tele.Tags.Operation().SetId(sp.SpanContext().TraceID().String())
	tele.Tags.Operation().SetParentId(pid)
	tele.Tags.Operation().SetName(sp.Name())

	exp.client.Track(&tele)
}

// Preprocesses the Otel span and dispatches it to app insights differently
// based on the span kind.
func (exp *AppInsightsExporter) Process(sp sdktrace.ReadOnlySpan) {
	success := true
	if sp.Status().Code != codes.Ok {
		success = false
	}

	attr := sp.Attributes()
	props := map[string]string{}
	for _, e := range attr {
		props[string(e.Key)] = e.Value.AsString()
	}

	switch sp.SpanKind() {
	case trace.SpanKindUnspecified:
		exp.ProcessInternal(sp, props)
	case trace.SpanKindInternal:
		exp.ProcessInternal(sp, props)
	case trace.SpanKindServer:
		exp.ProcessRequest(sp, success, props)
	case trace.SpanKindClient:
		exp.ProcessDependency(sp, success, props)
	case trace.SpanKindProducer:
		exp.ProcessDependency(sp, success, props)
	case trace.SpanKindConsumer:
		exp.ProcessEvent(sp, success, props)
	}
}

