package demo

import (
	"context"
	"encoding/json"
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

func TestKratosFlowClientGetFlowUsesAPIBaseAndForwardsOnlyOryCookies(t *testing.T) {
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
	req.Header.Set("Cookie", "a=b; ory_session=test; csrf=x")

	flow, err := client.GetFlow(context.Background(), req, "registration", "flow-123")
	if err != nil {
		t.Fatalf("GetFlow() error = %v", err)
	}
	if flow.ID != "flow-123" {
		t.Fatalf("unexpected flow: %+v", flow)
	}
	if gotCookie != "ory_session=test" {
		t.Fatalf("expected only Ory cookie to be forwarded, got %q", gotCookie)
	}
}

func TestFilterOryCookies(t *testing.T) {
	got := FilterOryCookies("foo=bar; ory_session=test; csrf=x; ory_kratos_session=y; csrf_token_abc=z")
	for _, want := range []string{"ory_session=test", "ory_kratos_session=y", "csrf_token_abc=z"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q to contain %q", got, want)
		}
	}
	if strings.Contains(got, "foo=bar") || strings.Contains(got, "csrf=x") {
		t.Fatalf("expected non-Ory cookies to be filtered, got %q", got)
	}
}

func TestSafeImageSrcRejectsUnsafeURL(t *testing.T) {
	if got := safeImageSrc("javascript:alert(1)"); got != "" {
		t.Fatalf("expected unsafe image URL to be rejected, got %q", got)
	}
	if got := safeImageSrc("data:text/html;base64,PHNjcmlwdA=="); got != "" {
		t.Fatalf("expected non-image data URL to be rejected, got %q", got)
	}
	if got := safeImageSrc("data:image/png;base64,abc"); got == "" {
		t.Fatal("expected image data URL to be allowed")
	}
}

func TestRenderPageRendersTOTPImageAndSecretText(t *testing.T) {
	rec := httptest.NewRecorder()

	err := RenderPage(rec, PageData{
		Title:       "Settings",
		Description: "Manage MFA",
		Flow: KratosFlow{
			UI: struct {
				Action   string          `json:"action"`
				Method   string          `json:"method"`
				Nodes    []KratosNode    `json:"nodes"`
				Messages []KratosMessage `json:"messages"`
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
	if !strings.Contains(body, "認証アプリ QR コード") {
		t.Fatalf("expected qr label in Japanese, got %s", body)
	}
	if !strings.Contains(body, "シークレットキー") {
		t.Fatalf("expected secret label in Japanese, got %s", body)
	}
	if !strings.Contains(body, "data:image/png;base64,abc") || !strings.Contains(body, "ABC123") {
		t.Fatalf("expected qr src and secret in body, got %s", body)
	}
	for _, fragment := range []string{"推しメンカラー", "#ffb2b2", "id=\"oshi-toggle\""} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected theme fragment %q, got %s", fragment, body)
		}
	}
	for _, fragment := range []string{"ログイン", "新規登録", "セキュリティ設定"} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected nav label %q, got %s", fragment, body)
		}
	}
	for _, fragment := range []string{`href="/"`, "復旧", "確認"} {
		if strings.Contains(body, fragment) {
			t.Fatalf("expected flow nav not to contain central portal link %q, got %s", fragment, body)
		}
	}
}

func TestRenderPageUsesAccountCenterDesignSystem(t *testing.T) {
	rec := httptest.NewRecorder()

	err := RenderPage(rec, PageData{
		Title:       "Login",
		Description: "Sign in",
		FlowType:    "login",
		OshiColor:   "#ffb2d8",
		Flow: func() KratosFlow {
			flow := KratosFlow{ID: "flow-123"}
			flow.UI.Action = "http://kratos/login"
			flow.UI.Method = "POST"
			return flow
		}(),
	})
	if err != nil {
		t.Fatalf("RenderPage() error = %v", err)
	}

	body := rec.Body.String()
	for _, fragment := range []string{
		"--oshi-weak:#fce7f3",
		"--surface-2:#fef9fd",
		"--radius-lg:32px",
		"brand-mark",
		"OshiLink",
		"推し色を選ぶ",
		`href="/recovery"`,
		"パスワードを忘れた場合",
		`href="/legal/terms"`,
		`href="/legal/privacy"`,
		"利用規約",
		"プライバシーポリシー",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected Kratos flow page to contain account center design fragment %q", fragment)
		}
	}
	for _, fragment := range []string{`href="/settings"`, "セキュリティ設定"} {
		if strings.Contains(body, fragment) {
			t.Fatalf("expected login page not to link to settings, got %s", body)
		}
	}
}

func TestRenderPageRendersPasskeyLoginTriggerButton(t *testing.T) {
	rec := httptest.NewRecorder()

	var flow KratosFlow
	if err := json.Unmarshal([]byte(`{
		"id": "flow-123",
		"ui": {
			"action": "http://kratos/login",
			"method": "POST",
			"nodes": [
				{
					"type": "script",
					"group": "passkey",
					"attributes": {
						"src": "/.well-known/ory/webauthn.js"
					}
				},
				{
					"type": "input",
					"group": "passkey",
					"attributes": {
						"name": "passkey_login_trigger",
						"type": "button",
						"value": "",
						"disabled": false,
						"onclickTrigger": "oryPasskeyLogin",
						"node_type": "input"
					},
					"meta": {
						"label": {
							"text": "Sign in with passkey"
						}
					}
				}
			]
		}
	}`), &flow); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	err := RenderPage(rec, PageData{
		Title:       "Login",
		Description: "Sign in",
		FlowType:    "login",
		Flow:        flow,
	})
	if err != nil {
		t.Fatalf("RenderPage() error = %v", err)
	}

	body := rec.Body.String()
	for _, fragment := range []string{
		`<button type="button" class="passkey-btn" name="passkey_login_trigger"`,
		`onclick="window.oryPasskeyLogin()"`,
		`>パスキーでログイン</button>`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected passkey login button fragment %q, got %s", fragment, body)
		}
	}
}

func TestRenderPageUsesLegalLinksOnRegistration(t *testing.T) {
	rec := httptest.NewRecorder()

	err := RenderPage(rec, PageData{
		Title:        "Registration",
		Description:  "Create account",
		FlowType:     "registration",
		LegalBaseURL: "https://auth.example.com",
		Flow: KratosFlow{
			UI: struct {
				Action   string          `json:"action"`
				Method   string          `json:"method"`
				Nodes    []KratosNode    `json:"nodes"`
				Messages []KratosMessage `json:"messages"`
			}{
				Action: "http://kratos/registration",
				Method: http.MethodPost,
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderPage() error = %v", err)
	}

	body := rec.Body.String()
	for _, fragment := range []string{
		`href="https://auth.example.com/legal/terms"`,
		`href="https://auth.example.com/legal/privacy"`,
		"利用規約",
		"プライバシーポリシー",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected registration page to contain legal fragment %q, got %s", fragment, body)
		}
	}
	for _, fragment := range []string{`href="/settings"`, "セキュリティ設定"} {
		if strings.Contains(body, fragment) {
			t.Fatalf("expected registration page not to link to settings, got %s", body)
		}
	}
}

func TestRenderPageSettingsOnlyLinksBackToAccountCenter(t *testing.T) {
	rec := httptest.NewRecorder()

	err := RenderPage(rec, PageData{
		Title:            "Settings",
		Description:      "Manage security settings",
		FlowType:         "settings",
		AccountCenterURL: "https://auth.example.com/account/",
		Flow: KratosFlow{
			UI: struct {
				Action   string          `json:"action"`
				Method   string          `json:"method"`
				Nodes    []KratosNode    `json:"nodes"`
				Messages []KratosMessage `json:"messages"`
			}{
				Action: "http://kratos/settings",
				Method: http.MethodPost,
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderPage() error = %v", err)
	}

	body := rec.Body.String()
	for _, fragment := range []string{
		`href="https://auth.example.com/account/"`,
		"アカウントセンター",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected settings page to contain account center fragment %q, got %s", fragment, body)
		}
	}
	for _, fragment := range []string{
		`href="/login"`,
		`href="/registration"`,
		"ログイン",
		"新規登録",
		"セキュリティ設定",
	} {
		if strings.Contains(body, fragment) {
			t.Fatalf("expected settings page not to contain auth nav fragment %q, got %s", fragment, body)
		}
	}
}

func TestRenderPageUsesEmailOnlyRegistration(t *testing.T) {
	rec := httptest.NewRecorder()

	err := RenderPage(rec, PageData{
		Title:       "Registration",
		Description: "Create account",
		FlowType:    "registration",
		Flow: KratosFlow{
			UI: struct {
				Action   string          `json:"action"`
				Method   string          `json:"method"`
				Nodes    []KratosNode    `json:"nodes"`
				Messages []KratosMessage `json:"messages"`
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
		`メールアドレスとパスワードを入力`,
		`8文字以上で、英大文字・英小文字・数字・記号のうち3種類以上を含めてください。`,
		`id="password-strength-panel"`,
		`input[name="traits.email"]`,
		`input[name="password"]`,
		`button type="submit" name="method" value="password"`,
		`hidden.value='email'`,
		`passwordField.setCustomValidity`,
		`passwordField.minLength=8`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected fragment %q, got %s", fragment, body)
		}
	}
	for _, fragment := range []string{
		`メールアドレスまたは電話番号`,
		`id="primary_identifier_display"`,
		`name="traits.phone"`,
		`hidden.value='phone'`,
		`電話番号として登録されます`,
	} {
		if strings.Contains(body, fragment) {
			t.Fatalf("expected email-only registration not to contain fragment %q, got %s", fragment, body)
		}
	}
}
