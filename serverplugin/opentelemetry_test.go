package serverplugin

import (
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 09:59
// @Version 1.0

func TestNewOpenTelemetryPlugin(t *testing.T) {
	type args struct {
		tracer      trace.Tracer
		propagators propagation.TextMapPropagator
	}
	tests := []struct {
		name string
		args args
		want *OpenTelemetryPlugin
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewOpenTelemetryPlugin(tt.args.tracer, tt.args.propagators); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOpenTelemetryPlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}
