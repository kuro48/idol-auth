package webhook

import (
	"net/http"
	"testing"
)

func TestValidateURLRejectsInsecureAndLocalTargets(t *testing.T) {
	tests := []string{
		"http://example.com/webhook",
		"https://localhost/webhook",
		"https://127.0.0.1/webhook",
		"https://[::1]/webhook",
		"https://10.0.0.5/webhook",
		"https://172.16.0.5/webhook",
		"https://192.168.1.5/webhook",
		"https://169.254.169.254/latest/meta-data",
		"https://user:pass@example.com/webhook",
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateURL(raw); err == nil {
				t.Fatalf("expected %q to be rejected", raw)
			}
		})
	}
}

func TestValidateURLAllowsPublicHTTPSURL(t *testing.T) {
	if err := ValidateURL("https://example.com/webhook"); err != nil {
		t.Fatalf("expected public https webhook URL to be valid: %v", err)
	}
}

func TestForbiddenWebhookIPClassification(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{ip: "127.0.0.1", want: true},
		{ip: "::1", want: true},
		{ip: "10.0.0.1", want: true},
		{ip: "172.16.0.1", want: true},
		{ip: "192.168.0.1", want: true},
		{ip: "169.254.169.254", want: true},
		{ip: "fc00::1", want: true},
		{ip: "8.8.8.8", want: false},
		{ip: "2001:4860:4860::8888", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got, err := isForbiddenWebhookIP(tt.ip)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("isForbiddenWebhookIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestWebhookHTTPClientDisablesProxyAndRejectsUnsafeRedirect(t *testing.T) {
	client := newHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected webhook client to use *http.Transport")
	}
	if transport.Proxy != nil {
		t.Fatal("expected webhook client to ignore environment proxies")
	}
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/webhook", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if err := client.CheckRedirect(req, nil); err == nil {
		t.Fatal("expected unsafe redirect target to be rejected")
	}
}
