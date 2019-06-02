# Reports

`apitest` includes a reporting mechanism that can generate sequence diagrams illustrating the inbound request, final response, any interactions with mocks and even database queries. You can even implement your own `ReportFormatter` to consume the report data which you might use to generate your own reports.

The key components in the reporting support are the

1. *Event* There are two types of events - HTTP events and Custom events. HTTP events represent mock interactions and the request into the application and final response. Custom events are used to generate data from arbitrary sources. A custom event contains a header and body. We use this event type within `apitest` to record database interactions.
1. *Recorder* records the events that happen during test execution, such as mock interactions, database interactions and HTTP interactions with the application under test. If you pass in your own recorder you can add custom events - then get a handle on those events by implementing the `ReportFormatter`. This might be useful for recording custom events generated from sources such as an Amazon S3 client.
1. *ReportFormatter* An interface that users can implement to generate custom reports. Receives the report recorder which exposes events. `SequenceDiagramFormatter` is an implementation of this interface included with `apitest` that a renders HTML sequence diagrams from event data.

## Sequence Diagrams

Configure the reporter to create sequence diagrams as follows

```go
apitest.New().
	Report(apitest.SequenceDiagram()).
	Handler(handler).
	Get("/user").
	Expect(t).
	Status(http.StatusOK).
	End()
```

In this [example](https://github.com/steinfletcher/apitest/tree/master/examples/sequence-diagrams) we implement a REST API and generate a sequence diagram with the http interactions.

The following diagram is generated which illustrates the interactions between collaborators in the test. The `sut` block is the system under test.

<span class="seqDiagIm">
![sequence diagram](/seq-diagram.png)
</span>

For each interaction, the http wire representation of the request/response is rendered in the event log below the diagram

<span class="eveLog">
![event log](/log.png)
</span>
