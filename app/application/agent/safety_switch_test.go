package agent

import (
	"benetnasch/app/domain/port"
	"context"
	stderrors "errors"
	"testing"
	"time"
)

type emergencySwitchCache struct {
	port.Cache
	value    string
	getErr   error
	setErr   error
	setValue string
}

func (c *emergencySwitchCache) Get(context.Context, string) (string, error) {
	if c.getErr != nil {
		return "", c.getErr
	}
	if c.value == "" {
		return "", port.ErrCacheMiss
	}
	return c.value, nil
}

func (c *emergencySwitchCache) Set(_ context.Context, _ string, value any, _ time.Duration) error {
	if c.setErr != nil {
		return c.setErr
	}
	c.setValue = value.(string)
	return nil
}

func TestEmergencySwitchDefaultsToConfiguredStateOnCacheMiss(t *testing.T) {
	for _, test := range []struct {
		name           string
		defaultStopped bool
		want           bool
	}{
		{name: "running by default", defaultStopped: false, want: false},
		{name: "stopped by default", defaultStopped: true, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			switcher, err := NewEmergencySwitch(&emergencySwitchCache{}, test.defaultStopped)
			if err != nil {
				t.Fatal(err)
			}
			got, err := switcher.IsStopped(context.Background())
			if err != nil || got != test.want {
				t.Fatalf("IsStopped() = %v, %v, want %v, nil", got, err, test.want)
			}
		})
	}
}

func TestEmergencySwitchFailsClosedForCorruptOrUnavailableState(t *testing.T) {
	corrupt, err := NewEmergencySwitch(&emergencySwitchCache{value: "unexpected"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if stopped, err := corrupt.IsStopped(context.Background()); err != nil || !stopped {
		t.Fatalf("corrupt switch = %v, %v, want stopped", stopped, err)
	}

	backendErr := stderrors.New("redis unavailable")
	unavailable, err := NewEmergencySwitch(&emergencySwitchCache{getErr: backendErr}, false)
	if err != nil {
		t.Fatal(err)
	}
	if stopped, err := unavailable.IsStopped(context.Background()); err == nil || !stopped {
		t.Fatalf("unavailable switch = %v, %v, want stopped and error", stopped, err)
	}
}

func TestEmergencySwitchPersistsExplicitState(t *testing.T) {
	cache := &emergencySwitchCache{}
	switcher, err := NewEmergencySwitch(cache, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := switcher.SetStopped(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if cache.setValue != "1" {
		t.Fatalf("stored stopped value = %q, want 1", cache.setValue)
	}
	if err := switcher.SetStopped(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if cache.setValue != "0" {
		t.Fatalf("stored running value = %q, want 0", cache.setValue)
	}
}

func TestEmergencySwitchPreservesContextErrors(t *testing.T) {
	switcher, err := NewEmergencySwitch(&emergencySwitchCache{getErr: context.Canceled, setErr: context.Canceled}, false)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := switcher.IsStopped(ctx); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("IsStopped() error = %v, want context canceled", err)
	}
	if err := switcher.SetStopped(ctx, true); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("SetStopped() error = %v, want context canceled", err)
	}
}
