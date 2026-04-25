package agent

import (
	"strings"
	"testing"
)

func TestParseObserverFlags_Defaults(t *testing.T) {
	cfg, err := ParseObserverFlags([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != 0 {
		t.Errorf("expected Interval=0, got %d", cfg.Interval)
	}
	if cfg.Quiet {
		t.Error("expected Quiet=false")
	}
}

func TestParseObserverFlags_Interval(t *testing.T) {
	cfg, err := ParseObserverFlags([]string{"--interval", "60"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != 60 {
		t.Errorf("expected Interval=60, got %d", cfg.Interval)
	}
}

func TestParseObserverFlags_Quiet(t *testing.T) {
	cfg, err := ParseObserverFlags([]string{"--quiet"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Quiet {
		t.Error("expected Quiet=true")
	}
}

func TestParseObserverFlags_InvalidInterval(t *testing.T) {
	_, err := ParseObserverFlags([]string{"--interval", "abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric interval")
	}
	if !strings.Contains(err.Error(), "positive number") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestParseObserverFlags_NegativeInterval(t *testing.T) {
	_, err := ParseObserverFlags([]string{"--interval", "-5"})
	if err == nil {
		t.Fatal("expected error for negative interval")
	}
}

func TestParseObserverFlags_Both(t *testing.T) {
	cfg, err := ParseObserverFlags([]string{"--quiet", "--interval", "30"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != 30 {
		t.Errorf("expected Interval=30, got %d", cfg.Interval)
	}
	if !cfg.Quiet {
		t.Error("expected Quiet=true")
	}
}

func TestBuildObserverOptionsMessage_NoOptions(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{})
	if msg != "" {
		t.Errorf("expected empty string, got: %s", msg)
	}
}

func TestBuildObserverOptionsMessage_IntervalOnly(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{Interval: 60})
	if !strings.Contains(msg, "Scan interval: 60 seconds") {
		t.Error("should contain interval")
	}
	if strings.Contains(msg, "Quiet mode") {
		t.Error("should not contain quiet mode")
	}
}

func TestBuildObserverOptionsMessage_QuietOnly(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{Quiet: true})
	if !strings.Contains(msg, "Quiet mode: enabled") {
		t.Error("should contain quiet mode")
	}
	if strings.Contains(msg, "Scan interval") {
		t.Error("should not contain interval")
	}
}

func TestBuildObserverOptionsMessage_Both(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{Interval: 30, Quiet: true})
	if !strings.Contains(msg, "Scan interval: 30 seconds") {
		t.Error("should contain interval")
	}
	if !strings.Contains(msg, "Quiet mode: enabled") {
		t.Error("should contain quiet mode")
	}
	if !strings.Contains(msg, "CLI Options provided:") {
		t.Error("should contain header")
	}
}
