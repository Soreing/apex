# AppInsights Exporter
Apex is a basic Open Telemetry span exporter to Azure App Insights, wrapped around the official SDK.

## Usage
Create an apex exporter that you can assign to tracers. You need an instrumentation key and a hook function to handle status messages from the AppInsights SDK.
```golang
instrKey := "12345678-1234-1234-1234-1234567890ab"
exp, err := apex.NewExporter(
	instrKey,
	func(msg string) error {
		fmt.Println(msg)
		return nil
	},
)
```

Submit a slice of Open Telemetry ReadOnlySpan objects to the exporter and they will be processed to extract key details before they are sent to the AppInsights SDK. Spans are typically created by the Open Telemetry SDK through using tracers.
```golang
spans := /* Slice of Read Only Spans*/
err := exp.ExportSpans(context.TODO(), spans)
if err != nil {
	panic(err)
}
```

To stop using the exporter, use the Shutdown method. The exporter will wait on the AppInsights SDK to submit its traces, and retry for up to a minute, or until the context is canceled. The shutdown function is typically called by the Open Telemetry SDK.
```golang
err := exp.Shutdown(context.TODO())
if err != nil {
	panic(err)
}
```

## Trace Attributes
The exporter automatically extracts information from the ReadOnlySpan objects to construct AppInsights traces. Some fields have default values that can be overridden with attributes on the ReadOnlySpan.
## Internal Events 

| Field | Source | Default |
|-------|--------|---------|
| Operation Id | Span Trace Id   | |
| Parent Id    | Span Parent Id  | |
| Event Time   | Span Start Time | |
| Name         | Span Name       | |
| Role         | Span Resource Service Name | "unknown-service" |

## Requests
| Field | Source | Default |
|-------|--------|---------|
| Operation Id | Span Trace Id       | |
| Parent Id    | Span Parent Id      | |
| Event Time   | Span Start Time     | |
| Name         | Span Name           | |
| Id           | Span Id             | |
| Duration     | Span End-Start Time | |
| Success      | Span Status         | |
| Role         | Span Resource Service Name     | "unknown-service" |
| Url          | Span "url" Attribute           | "" |
| ResponseCode | Span "responseCode" Attribute  | "" |

## Events
| Field | Source | Default |
|-------|--------|---------|
| Operation Id | Span Trace Id       | |
| Parent Id    | Span Parent Id      | |
| Event Time   | Span Start Time     | |
| Name         | Span Name           | |
| Id           | Span Id             | |
| Duration     | Span End-Start Time | |
| Success      | Span Status         | |
| Role         | Span Resource Service Name    | "unknown-service" |
| Url          | Span "key" Attribute          | "" |
| ResponseCode | Span "responseCode" Attribute | "" |

## Dependencies
| Field | Source | Default |
|-------|--------|---------|
| Operation Id | Span Trace Id       | |
| Parent Id    | Span Parent Id      | |
| Event Time   | Span Start Time     | |
| Name         | Span Name           | |
| Id           | Span Id             | |
| Duration     | Span End-Start Time | |
| Success      | Span Status         | |
| Role         | Span "source" Attribute    | "unknown-service" |
| Type         | Span "type" Attribute      | "" |
| Target       | Span Resource Service Name | "unknown-target" |



