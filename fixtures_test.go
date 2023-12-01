package apex

import (
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	trace "go.opentelemetry.io/otel/trace"
)

type mockTelemetryClient struct {
	appinsights.TelemetryClient
	tels     []appinsights.Telemetry
	closeDur time.Duration
}

func (tc *mockTelemetryClient) Track(tel appinsights.Telemetry) {
	tc.tels = append(tc.tels, tel)
}

func (tc *mockTelemetryClient) Channel() appinsights.TelemetryChannel {
	return &mockTelemetryChannel{closeDur: tc.closeDur}
}

type mockTelemetryChannel struct {
	appinsights.TelemetryChannel
	closeDur time.Duration
}

func (tc *mockTelemetryChannel) Close(t ...time.Duration) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		time.Sleep(tc.closeDur)
		ch <- struct{}{}
		close(ch)
	}()
	return ch
}

type mockSpan struct {
	sdktrace.ReadOnlySpan

	name   string
	kind   trace.SpanKind
	status sdktrace.Status

	startTime time.Time
	endTime   time.Time
	traceId   [16]byte
	parentId  [8]byte
	spanId    [8]byte

	res  *resource.Resource
	attr []attribute.KeyValue
}

func (s *mockSpan) Name() string {
	return s.name
}

func (s *mockSpan) SpanKind() trace.SpanKind {
	return s.kind
}

func (s *mockSpan) Status() sdktrace.Status {
	return s.status
}

func (s *mockSpan) StartTime() time.Time {
	return s.startTime
}

func (s *mockSpan) EndTime() time.Time {
	return s.endTime
}

func (s *mockSpan) Parent() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: s.traceId,
		SpanID:  s.parentId,
	})
}

func (s *mockSpan) SpanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: s.traceId,
		SpanID:  s.spanId,
	})
}

func (s *mockSpan) Resource() *resource.Resource {
	return s.res
}

func (s *mockSpan) Attributes() []attribute.KeyValue {
	return s.attr
}
