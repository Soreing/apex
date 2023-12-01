package apex

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	trace "go.opentelemetry.io/otel/trace"
)

// TestNewExporter tests that an exporter is created accurately given the
// input parameters are correct
func TestNewExporter(t *testing.T) {
	tests := []struct {
		Name   string
		IKey   string
		Logger func(msg string) error
		Error  error
	}{
		{
			Name:   "New exporter",
			IKey:   "",
			Logger: nil,
			Error:  nil,
		},
		{
			Name: "New exporter with logger",
			IKey: "",
			Logger: func(msg string) error {
				println(msg)
				return nil
			},
			Error: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			exp, err := NewExporter(test.IKey, test.Logger)

			assert.NotNil(t, exp)
			assert.Equal(t, test.Error, err)

			assert.NotNil(t, exp.mtx)
			assert.Equal(t, exp.closed, false)
			assert.Equal(t, test.IKey, exp.client.InstrumentationKey())
		})
	}
}

// TestExportSpans tests that spans fed to the exporter are processed
func TestExportSpans(t *testing.T) {
	tests := []struct {
		Name      string
		Exported  int
		Processed int
		Closed    bool
		Error     error
	}{
		{
			Name:      "Export nothing",
			Exported:  0,
			Processed: 0,
			Closed:    false,
			Error:     nil,
		},
		{
			Name:      "Export one span",
			Exported:  1,
			Processed: 1,
			Closed:    false,
			Error:     nil,
		},
		{
			Name:      "Export multiple spans",
			Exported:  5,
			Processed: 5,
			Closed:    false,
			Error:     nil,
		},
		{
			Name:      "Export after closed",
			Exported:  5,
			Processed: 0,
			Closed:    true,
			Error:     errors.New("exporter closed"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tcl := &mockTelemetryClient{}
			exp, _ := NewExporter("", nil)
			exp.client = tcl

			res, _ := resource.New(context.Background())

			spans := []sdktrace.ReadOnlySpan{}
			for i := 0; i < test.Exported; i++ {
				spans = append(spans, &mockSpan{
					kind:   trace.SpanKindUnspecified,
					status: sdktrace.Status{Code: codes.Ok},
					traceId: [16]byte{
						0x00, 0x11, 0x22, 0x33,
						0x44, 0x55, 0x66, 0x77,
						0x88, 0x99, 0xAA, 0xBB,
						0xCC, 0xDD, 0xEE, 0xFF,
					},
					parentId: [8]byte{
						0x01, 0x23, 0x45, 0x67,
						0x89, 0xAB, 0xCD, 0xEF,
					},
					spanId: [8]byte{
						0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x01,
					},
					res:  res,
					attr: []attribute.KeyValue{},
				})
			}

			if test.Closed {
				exp.closed = true
			}

			err := exp.ExportSpans(context.Background(), spans)

			assert.Equal(t, test.Error, err)
			assert.Equal(t, test.Processed, len(tcl.tels))
		})
	}
}

// TestExportSpans tests that spans fed to the exporter are processed
func TestShutdown(t *testing.T) {
	tests := []struct {
		Name     string
		Duration time.Duration
		Error    error
	}{
		{
			Name:     "Normal shutdown",
			Duration: time.Minute,
			Error:    nil,
		},
		{
			Name:     "Context canceled during shutdown",
			Duration: -time.Minute,
			Error:    errors.New("context canceled"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			exp, _ := NewExporter("", nil)

			ctx, cncl := context.WithDeadline(
				context.Background(),
				time.Now().Add(test.Duration),
			)

			err := exp.Shutdown(ctx)

			assert.Equal(t, test.Error, err)
			assert.Equal(t, true, exp.closed)
			cncl()
		})
	}
}

// TestProcessInternal tests that internal traces are processed accurately
func TestProcessInternal(t *testing.T) {
	tests := []struct {
		Name        string
		ParentId    [8]byte
		Kind        trace.SpanKind
		ResAttribs  []attribute.KeyValue
		SpanAttribs []attribute.KeyValue

		TelSource string
		TelParent string
		TelProps  map[string]string
	}{
		{
			Name:     "Process internal span type",
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindInternal,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("valid", "true"),
			},
			TelParent: "0123456789abcdef",
			TelSource: "test",
			TelProps: map[string]string{
				"valid": "true",
			},
		},
		{
			Name:     "Process unknown span type",
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindUnspecified,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
			},
			SpanAttribs: []attribute.KeyValue{},
			TelParent:   "0123456789abcdef",
			TelSource:   "test",
			TelProps:    map[string]string{},
		},
		{
			Name:        "Process internal span with no service name",
			ParentId:    [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:        trace.SpanKindInternal,
			ResAttribs:  []attribute.KeyValue{},
			SpanAttribs: []attribute.KeyValue{},
			TelParent:   "0123456789abcdef",
			TelSource:   "unknown-service",
			TelProps:    map[string]string{},
		},
		{
			Name:     "Process internal span with no parent",
			ParentId: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Kind:     trace.SpanKindInternal,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
			},
			SpanAttribs: []attribute.KeyValue{},
			TelParent:   "00112233445566778899aabbccddeeff",
			TelSource:   "test",
			TelProps:    map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tcl := &mockTelemetryClient{}
			exp, _ := NewExporter("", nil)
			exp.client = tcl

			res, _ := resource.New(
				context.Background(),
				resource.WithAttributes(test.ResAttribs...),
			)

			span := &mockSpan{
				name:      "span",
				kind:      test.Kind,
				status:    sdktrace.Status{Code: codes.Ok},
				startTime: time.Now(),
				traceId: [16]byte{
					0x00, 0x11, 0x22, 0x33,
					0x44, 0x55, 0x66, 0x77,
					0x88, 0x99, 0xAA, 0xBB,
					0xCC, 0xDD, 0xEE, 0xFF,
				},
				parentId: test.ParentId,
				spanId: [8]byte{
					0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x01,
				},
				res:  res,
				attr: test.SpanAttribs,
			}

			exp.process(span)

			assert.Equal(t, 1, len(tcl.tels))
			assert.IsType(t, tcl.tels[0], (*appinsights.EventTelemetry)(nil))
			tel := tcl.tels[0].(*appinsights.EventTelemetry)

			assert.Equal(t, span.name, tel.Name)
			assert.Equal(t, span.startTime, tel.Time())
			assert.Equal(t, test.TelSource, tel.ContextTags()["ai.cloud.role"])
			assert.Equal(t, test.TelParent, tel.ContextTags()["ai.operation.parentId"])
			assert.Equal(t, "00112233445566778899aabbccddeeff", tel.ContextTags()["ai.operation.id"])
			assert.Equal(t, test.TelProps, tel.GetProperties())
		})
	}
}

// TestProcessRequest tests that request traces are processed accurately
func TestProcessRequest(t *testing.T) {
	tests := []struct {
		Name        string
		Success     bool
		ParentId    [8]byte
		Kind        trace.SpanKind
		Duration    time.Duration
		ResAttribs  []attribute.KeyValue
		SpanAttribs []attribute.KeyValue

		TelId      string
		TelSource  string
		TelParent  string
		TelUrl     string
		TelResCode string
		TelProps   map[string]string
	}{
		{
			Name:     "Process successful request span",
			Success:  true,
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindServer,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("valid", "true"),
				attribute.String("url", "users/1234"),
				attribute.String("responseCode", "200"),
			},
			TelId:      "0000000000000001",
			TelParent:  "0123456789abcdef",
			TelSource:  "test",
			TelUrl:     "users/1234",
			TelResCode: "200",
			TelProps: map[string]string{
				"valid": "true",
			},
		},
		{
			Name:     "Process unsuccessful request span",
			Success:  false,
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindServer,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("valid", "true"),
				attribute.String("url", "users/abcd"),
				attribute.String("responseCode", "400"),
			},
			TelId:      "0000000000000001",
			TelParent:  "0123456789abcdef",
			TelSource:  "test",
			TelUrl:     "users/abcd",
			TelResCode: "400",
			TelProps: map[string]string{
				"valid": "true",
			},
		},
		{
			Name:        "Process request span with no props",
			Success:     true,
			ParentId:    [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:        trace.SpanKindServer,
			Duration:    time.Minute,
			ResAttribs:  []attribute.KeyValue{},
			SpanAttribs: []attribute.KeyValue{},
			TelId:       "0000000000000001",
			TelParent:   "0123456789abcdef",
			TelSource:   "unknown-service",
			TelUrl:      "",
			TelResCode:  "0",
			TelProps:    map[string]string{},
		},
		{
			Name:     "Process request span with no parent",
			Success:  true,
			ParentId: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Kind:     trace.SpanKindServer,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
				attribute.String("url", "users/1234"),
				attribute.String("responseCode", "200"),
			},
			SpanAttribs: []attribute.KeyValue{},
			TelId:       "0000000000000001",
			TelParent:   "00112233445566778899aabbccddeeff",
			TelSource:   "test",
			TelUrl:      "users/1234",
			TelResCode:  "200",
			TelProps:    map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tcl := &mockTelemetryClient{}
			exp, _ := NewExporter("", nil)
			exp.client = tcl

			res, _ := resource.New(
				context.Background(),
				resource.WithAttributes(test.ResAttribs...),
			)

			status := sdktrace.Status{Code: codes.Error}
			if test.Success {
				status = sdktrace.Status{Code: codes.Ok}
			}

			now := time.Now()
			span := &mockSpan{
				name:      "span",
				kind:      test.Kind,
				status:    status,
				startTime: now,
				endTime:   now.Add(time.Minute),
				traceId: [16]byte{
					0x00, 0x11, 0x22, 0x33,
					0x44, 0x55, 0x66, 0x77,
					0x88, 0x99, 0xAA, 0xBB,
					0xCC, 0xDD, 0xEE, 0xFF,
				},
				parentId: test.ParentId,
				spanId: [8]byte{
					0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x01,
				},
				res:  res,
				attr: test.SpanAttribs,
			}

			exp.process(span)

			assert.Equal(t, 1, len(tcl.tels))
			assert.IsType(t, tcl.tels[0], (*appinsights.RequestTelemetry)(nil))
			tel := tcl.tels[0].(*appinsights.RequestTelemetry)

			assert.Equal(t, test.TelId, tel.Id)
			assert.Equal(t, span.name, tel.Name)
			assert.Equal(t, span.startTime, tel.Time())
			assert.Equal(t, test.Duration, tel.Duration)
			assert.Equal(t, test.Success, tel.Success)
			assert.Equal(t, test.TelUrl, tel.Url)
			assert.Equal(t, test.TelResCode, tel.ResponseCode)
			assert.Equal(t, test.TelSource, tel.ContextTags()["ai.cloud.role"])
			assert.Equal(t, test.TelParent, tel.ContextTags()["ai.operation.parentId"])
			assert.Equal(t, "00112233445566778899aabbccddeeff", tel.ContextTags()["ai.operation.id"])
			assert.Equal(t, test.TelProps, tel.GetProperties())
		})
	}
}

// TestProcessEvent tests that event consumer traces are processed accurately
func TestProcessEvent(t *testing.T) {
	tests := []struct {
		Name        string
		Success     bool
		ParentId    [8]byte
		Kind        trace.SpanKind
		Duration    time.Duration
		ResAttribs  []attribute.KeyValue
		SpanAttribs []attribute.KeyValue

		TelId      string
		TelSource  string
		TelParent  string
		TelUrl     string
		TelResCode string
		TelProps   map[string]string
	}{
		{
			Name:     "Process successful event span",
			Success:  true,
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindConsumer,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("valid", "true"),
				attribute.String("key", "service.messages.created"),
				attribute.String("responseCode", "200"),
			},
			TelId:      "0000000000000001",
			TelParent:  "0123456789abcdef",
			TelSource:  "test",
			TelUrl:     "service.messages.created",
			TelResCode: "200",
			TelProps: map[string]string{
				"valid": "true",
			},
		},
		{
			Name:     "Process unsuccessful request span",
			Success:  false,
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindConsumer,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("valid", "true"),
				attribute.String("key", "service.messages.created"),
				attribute.String("responseCode", "400"),
			},
			TelId:      "0000000000000001",
			TelParent:  "0123456789abcdef",
			TelSource:  "test",
			TelUrl:     "service.messages.created",
			TelResCode: "400",
			TelProps: map[string]string{
				"valid": "true",
			},
		},
		{
			Name:        "Process request span with no props",
			Success:     true,
			ParentId:    [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:        trace.SpanKindConsumer,
			Duration:    time.Minute,
			ResAttribs:  []attribute.KeyValue{},
			SpanAttribs: []attribute.KeyValue{},
			TelId:       "0000000000000001",
			TelParent:   "0123456789abcdef",
			TelSource:   "unknown-service",
			TelUrl:      "",
			TelResCode:  "0",
			TelProps:    map[string]string{},
		},
		{
			Name:     "Process request span with no parent",
			Success:  true,
			ParentId: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Kind:     trace.SpanKindConsumer,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test"),
				attribute.String("key", "service.messages.created"),
				attribute.String("responseCode", "200"),
			},
			SpanAttribs: []attribute.KeyValue{},
			TelId:       "0000000000000001",
			TelParent:   "00112233445566778899aabbccddeeff",
			TelSource:   "test",
			TelUrl:      "service.messages.created",
			TelResCode:  "200",
			TelProps:    map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tcl := &mockTelemetryClient{}
			exp, _ := NewExporter("", nil)
			exp.client = tcl

			res, _ := resource.New(
				context.Background(),
				resource.WithAttributes(test.ResAttribs...),
			)

			status := sdktrace.Status{Code: codes.Error}
			if test.Success {
				status = sdktrace.Status{Code: codes.Ok}
			}

			now := time.Now()
			span := &mockSpan{
				name:      "span",
				kind:      test.Kind,
				status:    status,
				startTime: now,
				endTime:   now.Add(time.Minute),
				traceId: [16]byte{
					0x00, 0x11, 0x22, 0x33,
					0x44, 0x55, 0x66, 0x77,
					0x88, 0x99, 0xAA, 0xBB,
					0xCC, 0xDD, 0xEE, 0xFF,
				},
				parentId: test.ParentId,
				spanId: [8]byte{
					0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x01,
				},
				res:  res,
				attr: test.SpanAttribs,
			}

			exp.process(span)

			assert.Equal(t, 1, len(tcl.tels))
			assert.IsType(t, tcl.tels[0], (*appinsights.RequestTelemetry)(nil))
			tel := tcl.tels[0].(*appinsights.RequestTelemetry)

			assert.Equal(t, test.TelId, tel.Id)
			assert.Equal(t, span.name, tel.Name)
			assert.Equal(t, span.startTime, tel.Time())
			assert.Equal(t, test.Duration, tel.Duration)
			assert.Equal(t, test.Success, tel.Success)
			assert.Equal(t, test.TelUrl, tel.Url)
			assert.Equal(t, test.TelResCode, tel.ResponseCode)
			assert.Equal(t, test.TelSource, tel.ContextTags()["ai.cloud.role"])
			assert.Equal(t, test.TelParent, tel.ContextTags()["ai.operation.parentId"])
			assert.Equal(t, "00112233445566778899aabbccddeeff", tel.ContextTags()["ai.operation.id"])
			assert.Equal(t, test.TelProps, tel.GetProperties())
		})
	}
}

// TestProcessDependency tests that dependency traces are processed accurately
func TestProcessDependency(t *testing.T) {
	tests := []struct {
		Name        string
		Success     bool
		ParentId    [8]byte
		Kind        trace.SpanKind
		Duration    time.Duration
		ResAttribs  []attribute.KeyValue
		SpanAttribs []attribute.KeyValue

		TelId     string
		TelType   string
		TelSource string
		TelTarget string
		TelParent string
		TelProps  map[string]string
	}{
		{
			Name:     "Process successful client dependency span",
			Success:  true,
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindClient,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("client"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("valid", "true"),
				attribute.String("source", "server"),
				attribute.String("type", "httpclient"),
			},
			TelId:     "0000000000000001",
			TelParent: "0123456789abcdef",
			TelSource: "server",
			TelTarget: "client",
			TelType:   "httpclient",
			TelProps: map[string]string{
				"valid": "true",
			},
		},
		{
			Name:     "Process successful producer dependency span",
			Success:  true,
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindProducer,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("queue"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("valid", "true"),
				attribute.String("source", "server"),
				attribute.String("type", "event"),
			},
			TelId:     "0000000000000001",
			TelParent: "0123456789abcdef",
			TelSource: "server",
			TelTarget: "queue",
			TelType:   "event",
			TelProps: map[string]string{
				"valid": "true",
			},
		},
		{
			Name:     "Process unsuccessful client dependency span",
			Success:  false,
			ParentId: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:     trace.SpanKindClient,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("client"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("source", "server"),
				attribute.String("type", "httpclient"),
			},
			TelId:     "0000000000000001",
			TelParent: "0123456789abcdef",
			TelSource: "server",
			TelTarget: "client",
			TelType:   "httpclient",
			TelProps:  map[string]string{},
		},
		{
			Name:        "Process client dependency span with no props",
			Success:     false,
			ParentId:    [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
			Kind:        trace.SpanKindClient,
			Duration:    time.Minute,
			ResAttribs:  []attribute.KeyValue{},
			SpanAttribs: []attribute.KeyValue{},
			TelId:       "0000000000000001",
			TelParent:   "0123456789abcdef",
			TelSource:   "unknown-service",
			TelTarget:   "unknown-target",
			TelType:     "",
			TelProps:    map[string]string{},
		},
		{
			Name:     "Process client dependency span with no parent",
			Success:  true,
			ParentId: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Kind:     trace.SpanKindClient,
			Duration: time.Minute,
			ResAttribs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("client"),
			},
			SpanAttribs: []attribute.KeyValue{
				attribute.String("source", "server"),
				attribute.String("type", "httpclient"),
			},
			TelId:     "0000000000000001",
			TelParent: "00112233445566778899aabbccddeeff",
			TelSource: "server",
			TelTarget: "client",
			TelType:   "httpclient",
			TelProps:  map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tcl := &mockTelemetryClient{}
			exp, _ := NewExporter("", nil)
			exp.client = tcl

			res, _ := resource.New(
				context.Background(),
				resource.WithAttributes(test.ResAttribs...),
			)

			status := sdktrace.Status{Code: codes.Error}
			if test.Success {
				status = sdktrace.Status{Code: codes.Ok}
			}

			now := time.Now()
			span := &mockSpan{
				name:      "span",
				kind:      test.Kind,
				status:    status,
				startTime: now,
				endTime:   now.Add(time.Minute),
				traceId: [16]byte{
					0x00, 0x11, 0x22, 0x33,
					0x44, 0x55, 0x66, 0x77,
					0x88, 0x99, 0xAA, 0xBB,
					0xCC, 0xDD, 0xEE, 0xFF,
				},
				parentId: test.ParentId,
				spanId: [8]byte{
					0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x01,
				},
				res:  res,
				attr: test.SpanAttribs,
			}

			exp.process(span)

			assert.Equal(t, 1, len(tcl.tels))
			assert.IsType(t, tcl.tels[0], (*appinsights.RemoteDependencyTelemetry)(nil))
			tel := tcl.tels[0].(*appinsights.RemoteDependencyTelemetry)

			assert.Equal(t, test.TelId, tel.Id)
			assert.Equal(t, span.name, tel.Name)
			assert.Equal(t, span.startTime, tel.Time())
			assert.Equal(t, test.Duration, tel.Duration)
			assert.Equal(t, test.Success, tel.Success)
			assert.Equal(t, test.TelTarget, tel.Target)
			assert.Equal(t, test.TelType, tel.Type)
			assert.Equal(t, test.TelSource, tel.ContextTags()["ai.cloud.role"])
			assert.Equal(t, test.TelParent, tel.ContextTags()["ai.operation.parentId"])
			assert.Equal(t, "00112233445566778899aabbccddeeff", tel.ContextTags()["ai.operation.id"])
			assert.Equal(t, test.TelProps, tel.GetProperties())
		})
	}
}
