package sandbox

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// --provision-timeout replaces a fixed ten minutes that a slow apt upgrade
// overran. No flag means the default; a flag must be a positive duration.
func TestParseProvisionTimeout(t *testing.T) {
	t.Run("defaults to thirty minutes", func(t *testing.T) {
		for _, s := range []string{"", "  "} {
			got, err := ParseProvisionTimeout(s)
			if err != nil {
				t.Fatalf("%q: %v", s, err)
			}
			if got != 30*time.Minute {
				t.Errorf("%q: got %s, want 30m", s, got)
			}
		}
	})

	t.Run("reads a Go duration", func(t *testing.T) {
		got, err := ParseProvisionTimeout("45m")
		if err != nil {
			t.Fatal(err)
		}
		if got != 45*time.Minute {
			t.Errorf("got %s, want 45m", got)
		}
	})

	t.Run("refuses what is not a positive duration", func(t *testing.T) {
		for _, s := range []string{"soon", "10", "0", "-5m"} {
			if _, err := ParseProvisionTimeout(s); err == nil {
				t.Errorf("%q: expected an error", s)
			} else if !strings.Contains(err.Error(), "--provision-timeout") {
				t.Errorf("%q: error %q should name the flag", s, err)
			}
		}
	})
}

// The probe is one SSH round trip whose output the wait reads back. A failed
// connection returns nothing at all, which must read as "nothing known yet"
// rather than as any decision. The errors of cloud-init come as a YAML list,
// inline or one item on each line.
func TestParseProvisionProbe(t *testing.T) {
	got := parseProvisionProbe("marker=no\nlog=Get:12 https://deb.nodesource.com/node_22.x nodistro InRelease\ncloud-init: status: running\ncloud-init: errors: []\n")
	want := provisionProbe{cloudInit: "running", status: "status: running\nerrors: []", lastLog: "Get:12 https://deb.nodesource.com/node_22.x nodistro InRelease"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	if got := parseProvisionProbe("marker=yes\nlog=\ncloud-init: status: done\n"); !got.marker {
		t.Errorf("marker=yes should read as provisioned: %+v", got)
	}

	for _, errs := range []string{"cloud-init: errors: ['scripts_user failed']\n", "cloud-init: errors:\ncloud-init: \t- scripts_user failed\ncloud-init: recoverable_errors: {}\n"} {
		if got := parseProvisionProbe("marker=no\ncloud-init: status: done\n" + errs); !got.hasErrors {
			t.Errorf("%q should read as errors: %+v", errs, got)
		}
	}

	if got := parseProvisionProbe(""); got != (provisionProbe{}) {
		t.Errorf("a failed connection should read as unknown, got %+v", got)
	}
}

// Only a real cloud-init failure stops the wait on the first probe, and the
// message carries the status and the last log line. "done" without the
// marker gets the grace period, then stops with its own message.
func TestWaitForMarkerStopsOnCloudInitFailure(t *testing.T) {
	quietWait(t)

	for name, probe := range map[string]provisionProbe{
		"error":  {cloudInit: "error", status: "status: error", lastLog: "E: Unable to locate package nodejs"},
		"errors": {cloudInit: "done", hasErrors: true, status: "status: done\nerrors: [x]", lastLog: "Cloud-init v. 24.4 finished"},
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			err := waitForMarker(time.Now().Add(time.Hour), func() provisionProbe {
				calls++
				return probe
			})
			if err == nil {
				t.Fatal("expected the wait to stop")
			}
			if calls != 1 {
				t.Errorf("probed %d times; a cloud-init failure should stop the wait on the first probe", calls)
			}
			for _, want := range []string{provisionedMarker, probe.status, probe.lastLog} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q should contain %q", err, want)
				}
			}
		})
	}

	t.Run("done without the marker after the grace period", func(t *testing.T) {
		markerGrace = 20 * time.Millisecond
		calls := 0
		err := waitForMarker(time.Now().Add(time.Hour), func() provisionProbe {
			calls++
			return provisionProbe{cloudInit: "done", status: "status: done", lastLog: "finished"}
		})
		if err == nil || !strings.Contains(err.Error(), "finished without writing") || !strings.Contains(err.Error(), "status: done") {
			t.Fatalf("error = %v, want the finished-without-marker message with the status", err)
		}
		if calls < 2 {
			t.Errorf("probed %d times; done without the marker should poll through the grace period", calls)
		}
	})
}

// A marker that appears, or a deadline that passes, are the two ways the
// wait ends while cloud-init is still running.
func TestWaitForMarkerReturnsOnMarkerOrDeadline(t *testing.T) {
	quietWait(t)

	t.Run("returns once the marker appears", func(t *testing.T) {
		calls := 0
		err := waitForMarker(time.Now().Add(time.Hour), func() provisionProbe {
			calls++
			return provisionProbe{marker: calls >= 3, cloudInit: "running"}
		})
		if err != nil {
			t.Fatal(err)
		}
		if calls != 3 {
			t.Errorf("probed %d times, want 3", calls)
		}
	})

	t.Run("reports the timeout and the last log line", func(t *testing.T) {
		err := waitForMarker(time.Now().Add(20*time.Millisecond), func() provisionProbe {
			return provisionProbe{cloudInit: "running", lastLog: "Setting up nodejs"}
		})
		if err == nil {
			t.Fatal("expected a timeout")
		}
		for _, want := range []string{"did not complete", "Setting up nodejs"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q should contain %q", err, want)
			}
		}
	})
}

// A long wait must read as alive: at most once a report interval, the wait
// says how long it has been and what cloud-init last logged.
func TestWaitForMarkerReportsProgress(t *testing.T) {
	quietWait(t)
	provisionReportInterval = 0

	stderr := captureStderr(t)
	calls := 0
	err := waitForMarker(time.Now().Add(time.Hour), func() provisionProbe {
		calls++
		return provisionProbe{marker: calls >= 2, cloudInit: "running", lastLog: "Unpacking docker-ce"}
	})
	out := stderr()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Still provisioning after") || !strings.Contains(out, "Unpacking docker-ce") {
		t.Errorf("progress line should say how long and what cloud-init logged, got %q", out)
	}
}

// Create asks for the timeout through one seam, so the flag reaches the wait.
func TestCreatePassesTheProvisionTimeoutToTheWait(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)

	var got time.Duration
	orig := waitReady
	waitReady = func(ip, keyPath, name, user string, timeout time.Duration) error {
		got = timeout
		return errors.New("stop here")
	}
	t.Cleanup(func() { waitReady = orig })

	p := &recordingProvider{ip: "5.6.7.8", dropletID: 55}
	if err := Create(p, CreateOptions{Name: "slow", ProvisionTimeout: 45 * time.Minute}); err == nil {
		t.Fatal("expected the stubbed wait to stop the create")
	}
	if got != 45*time.Minute {
		t.Errorf("wait received %s, want 45m", got)
	}
}

// quietWait removes the real delays from the wait loop for one test.
func quietWait(t *testing.T) {
	t.Helper()
	origPoll, origReport, origGrace := provisionPollInterval, provisionReportInterval, markerGrace
	provisionPollInterval = 0
	t.Cleanup(func() { provisionPollInterval, provisionReportInterval, markerGrace = origPoll, origReport, origGrace })
}
