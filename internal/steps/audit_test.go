package steps

import "testing"

func TestAuditResultInvariants(t *testing.T) {
	results := Audit()
	if len(results) < 12 {
		t.Fatalf("expected at least 12 checks, got %d", len(results))
	}
	seen := map[string]bool{}
	for _, r := range results {
		switch r.Status {
		case "PASS", "WARN", "INFO":
		default:
			t.Errorf("%s: invalid status %q", r.Name, r.Status)
		}
		if r.Status == "WARN" && r.Severity == "" {
			t.Errorf("%s: WARN must carry a severity", r.Name)
		}
		if r.Status != "WARN" && r.Severity != "" {
			t.Errorf("%s: severity %q on non-WARN status %s", r.Name, r.Severity, r.Status)
		}
		if seen[r.Name] {
			t.Errorf("duplicate check name %q", r.Name)
		}
		seen[r.Name] = true
		if r.Detail == "" {
			t.Errorf("%s: empty detail", r.Name)
		}
	}
}

func TestDaemonTLSChecks(t *testing.T) {
	cases := []struct {
		name   string
		cfg    map[string]any
		status string
	}{
		{"no hosts", map[string]any{}, "PASS"},
		{"tcp without tls", map[string]any{"hosts": []any{"tcp://0.0.0.0:2375"}}, "WARN"},
		{"tcp with tlsverify", map[string]any{"hosts": []any{"tcp://0.0.0.0:2376"}, "tlsverify": true}, "PASS"},
		{"unix socket only", map[string]any{"hosts": []any{"unix:///var/run/docker.sock"}}, "PASS"},
	}
	for _, c := range cases {
		if got := checkDaemonTLS(c.cfg).Status; got != c.status {
			t.Errorf("%s: got %s, want %s", c.name, got, c.status)
		}
	}
}

func TestInsecureRegistries(t *testing.T) {
	r := checkInsecureRegistries(map[string]any{"insecure-registries": []any{"reg.local:5000"}})
	if r.Status != "WARN" || r.Severity != "high" {
		t.Errorf("expected WARN/high, got %s/%s", r.Status, r.Severity)
	}
	if checkInsecureRegistries(map[string]any{}).Status != "PASS" {
		t.Error("empty config should PASS")
	}
}
