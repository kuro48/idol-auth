package demo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKratosFlowClientBrowserInitURLUsesBrowserBase(t *testing.T) {
	client := NewKratosFlowClient("http://kratos:4433", "http://localhost:4433")

	got := client.BrowserInitURL("registration")

	if got != "http://localhost:4433/self-service/registration/browser" {
		t.Fatalf("unexpected browser init url: %q", got)
	}
}

func TestKratosFlowClientGetFlowUsesAPIBaseAndForwardsCookie(t *testing.T) {
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		if r.URL.Path != "/self-service/registration/flows" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "flow-123" {
			t.Fatalf("unexpected flow id: %q", r.URL.Query().Get("id"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"flow-123","ui":{"action":"http://kratos/submit","method":"POST"}}`))
	}))
	defer srv.Close()

	client := NewKratosFlowClient(srv.URL, "http://localhost:4433")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cookie", "a=b")

	flow, err := client.GetFlow(context.Background(), req, "registration", "flow-123")
	if err != nil {
		t.Fatalf("GetFlow() error = %v", err)
	}
	if flow.ID != "flow-123" {
		t.Fatalf("unexpected flow: %+v", flow)
	}
	if gotCookie != "a=b" {
		t.Fatalf("expected cookie to be forwarded, got %q", gotCookie)
	}
}

func TestRenderPageRendersTOTPImageAndSecretText(t *testing.T) {
	rec := httptest.NewRecorder()

	err := RenderPage(rec, PageData{
		Title:       "Settings",
		Description: "Manage MFA",
		Flow: KratosFlow{
			UI: struct {
				Action string       `json:"action"`
				Method string       `json:"method"`
				Nodes  []KratosNode `json:"nodes"`
			}{
				Action: "http://kratos/settings",
				Method: http.MethodPost,
				Nodes: []KratosNode{
					{
						Type:  "img",
						Group: "totp",
						Meta: KratosNodeMeta{Label: &struct {
							Text string `json:"text"`
						}{Text: "Authenticator app QR code"}},
						Attributes: KratosAttributes{Src: "data:image/png;base64,abc"},
					},
					{
						Type:  "text",
						Group: "totp",
						Meta: KratosNodeMeta{Label: &struct {
							Text string `json:"text"`
						}{Text: "Authenticator secret"}},
						Attributes: KratosAttributes{Text: &struct {
							ID      int    `json:"id"`
							Text    string `json:"text"`
							Type    string `json:"type"`
							Context struct {
								Secret string `json:"secret"`
							} `json:"context"`
						}{Text: "Authenticator secret", Context: struct {
							Secret string `json:"secret"`
						}{Secret: "ABC123"}}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderPage() error = %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Authenticator app QR code") {
		t.Fatalf("expected qr label, got %s", body)
	}
	if !strings.Contains(body, "Authenticator secret") {
		t.Fatalf("expected secret label, got %s", body)
	}
	if !strings.Contains(body, "data:image/png;base64,abc") || !strings.Contains(body, "ABC123") {
		t.Fatalf("expected qr src and secret in body, got %s", body)
	}
	for _, fragment := range []string{"推しメンカラー", "#ffb2b2", "id=\"oshi-toggle\""} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected theme fragment %q, got %s", fragment, body)
		}
	}
	for _, fragment := range []string{"ホーム", "ログイン", "新規登録", "復旧", "確認", "設定"} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected nav label %q, got %s", fragment, body)
		}
	}
}

func TestRenderPageHidesPrimaryIdentifierTypeAndAutoInfersOnRegistration(t *testing.T) {
	rec := httptest.NewRecorder()

	err := RenderPage(rec, PageData{
		Title:       "Registration",
		Description: "Create account",
		FlowType:    "registration",
		Flow: KratosFlow{
			UI: struct {
				Action string       `json:"action"`
				Method string       `json:"method"`
				Nodes  []KratosNode `json:"nodes"`
			}{
				Action: "http://kratos/registration",
				Method: http.MethodPost,
				Nodes: []KratosNode{
					{
						Meta: KratosNodeMeta{Label: &struct {
							Text string `json:"text"`
						}{Text: "Primary Identifier Type"}},
						Attributes: KratosAttributes{
							Name:  "traits.primary_identifier_type",
							Type:  "text",
							Value: "email",
						},
					},
					{
						Meta: KratosNodeMeta{Label: &struct {
							Text string `json:"text"`
						}{Text: "Email"}},
						Attributes: KratosAttributes{
							Name: "traits.email",
							Type: "email",
						},
					},
					{
						Meta: KratosNodeMeta{Label: &struct {
							Text string `json:"text"`
						}{Text: "Phone Number"}},
						Attributes: KratosAttributes{
							Name: "traits.phone",
							Type: "tel",
						},
					},
					{
						Meta: KratosNodeMeta{Label: &struct {
							Text string `json:"text"`
						}{Text: "Password"}},
						Attributes: KratosAttributes{
							Name:     "password",
							Type:     "password",
							Required: true,
						},
					},
					{
						Meta: KratosNodeMeta{Label: &struct {
							Text string `json:"text"`
						}{Text: "Sign up"}},
						Attributes: KratosAttributes{
							Name:  "method",
							Type:  "submit",
							Value: "profile",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderPage() error = %v", err)
	}

	body := rec.Body.String()
	if strings.Contains(body, `<label for="traits.primary_identifier_type">`) {
		t.Fatalf("expected primary identifier type field to be hidden, got %s", body)
	}
	for _, fragment := range []string{
		`name="traits.primary_identifier_type"`,
		`type="hidden"`,
		`data-flow-type="registration"`,
		`メールアドレスまたは電話番号とパスワードを同じ画面で入力`,
		`8文字以上で、英大文字・英小文字・数字・記号のうち3種類以上を含めてください。`,
		`id="password-strength-panel"`,
		`input[name="traits.email"]`,
		`input[name="traits.phone"]`,
		`input[name="password"]`,
		`button type="submit" name="method" value="password"`,
		`hidden.value='email'`,
		`hidden.value='phone'`,
		`passwordField.setCustomValidity`,
		`passwordField.minLength=8`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected fragment %q, got %s", fragment, body)
		}
	}
}
