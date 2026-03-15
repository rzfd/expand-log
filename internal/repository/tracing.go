package repository

import (
	"context"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var repositoryTracer = otel.Tracer("repository")
var whitespaceRE = regexp.MustCompile(`\s+`)

func startRepositorySpan(ctx context.Context, operation string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return repositoryTracer.Start(ctx, operation, trace.WithAttributes(attrs...))
}

func startDBSpan(ctx context.Context, operation string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	base := []attribute.KeyValue{
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", strings.ToUpper(operation)),
	}
	base = append(base, attrs...)
	return repositoryTracer.Start(ctx, "db."+strings.ToLower(operation), trace.WithAttributes(base...))
}

func setDBStatement(span trace.Span, statement string) {
	sanitized := whitespaceRE.ReplaceAllString(strings.TrimSpace(statement), " ")
	if sanitized == "" {
		return
	}
	span.SetAttributes(attribute.String("db.statement", sanitized))
}

func markSpanError(span trace.Span, err error, message string) {
	if err != nil {
		span.RecordError(err)
		if code := PgErrorCode(err); code != "" {
			span.SetAttributes(attribute.String("db.error.code", code))
		}
		if kind := PgErrorKind(err); kind != "" {
			span.SetAttributes(attribute.String("db.error.kind", kind))
		}
	}
	span.SetStatus(codes.Error, message)
}
