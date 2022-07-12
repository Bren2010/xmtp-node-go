// Package tracing enables [Datadog APM tracing](https://docs.datadoghq.com/tracing/) capabilities,
// focusing specifically on [Error Tracking](https://docs.datadoghq.com/tracing/error_tracking/)
package tracing

import (
	"context"

	"go.uber.org/zap"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var (
	// reimport relevant bits of the tracer API
	StartSpanFromContext = tracer.StartSpanFromContext
	WithError            = tracer.WithError
)

type logger struct{ *zap.Logger }

func (l logger) Log(msg string) {
	l.Error(msg)
}

// Start boots the datadog tracer, run this once early in the startup sequence.
func Start(l *zap.Logger) {
	tracer.Start(tracer.WithService("xmtp-node"), tracer.WithLogger(logger{l}))
}

// Stop shuts down the datadog tracer, defer this right after Start().
func Stop() {
	tracer.Stop()
}

// Do executes action in the context of a top level span,
// tagging the span with the error if the action panics.
// This should trigger DD APM's Error Tracking to record the error.
func Do(ctx context.Context, spanName string, action func(context.Context)) {
	span, ctx := tracer.StartSpanFromContext(ctx, spanName)
	defer func() {
		r := recover()
		switch r := r.(type) {
		case error:
			// If action panics with an error,
			// finish the span with the error.
			span.Finish(WithError(r))
		default:
			// This is the normal non-panicking path
			// as well as path with panics that don't have an error.
			span.Finish()
		}
		if r != nil {
			// Repanic so that we don't suppress normal panic behavior.
			panic(r)
		}
	}()
	action(ctx)
}

// Link connects a logger to a particular trace and span.
// DD APM should provide some additional functionality based on that.
func Link(span tracer.Span, l *zap.Logger) *zap.Logger {
	return l.With(
		zap.Uint64("dd.trace_id", span.Context().TraceID()),
		zap.Uint64("dd.span_id", span.Context().SpanID()))
}