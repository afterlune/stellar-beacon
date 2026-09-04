package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

type operationalMetricsProviderFake struct {
	snapshot port.OperationalMetricsSnapshot
}

func (f operationalMetricsProviderFake) Snapshot(context.Context) port.OperationalMetricsSnapshot {
	return f.snapshot
}

func TestOperationalObservabilityServiceReturnsSanitizedSnapshotWhenEnabled(t *testing.T) {
	want := port.OperationalMetricsSnapshot{GeneratedAt: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)}
	service := NewOperationalObservabilityService(operationalMetricsProviderFake{snapshot: want}, true)

	result := service.Get(context.Background())
	if !result.Flag || !reflect.DeepEqual(result.Data, want) {
		t.Fatalf("observability result = %+v, want successful snapshot", result)
	}
}

func TestOperationalObservabilityServiceFailsClosedWhenDisabled(t *testing.T) {
	service := NewOperationalObservabilityService(operationalMetricsProviderFake{}, false)
	result := service.Get(context.Background())
	if result.Flag || result.Message != "运行时观测暂未开启" {
		t.Fatalf("disabled observability result = %+v", result)
	}
}

func TestOperationalObservabilityServiceReportsMissingProvider(t *testing.T) {
	service := NewOperationalObservabilityService(nil, true)
	result := service.Get(context.Background())
	if result.Flag || result.Message == "" {
		t.Fatalf("missing provider result = %+v", result)
	}
}
