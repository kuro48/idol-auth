package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type KratosFlow struct {
	ID string `json:"id"`
	UI struct {
		Action string       `json:"action"`
		Method string       `json:"method"`
		Nodes  []KratosNode `json:"nodes"`
	} `json:"ui"`
	Messages []KratosMessage `json:"messages"`
}

type KratosNode struct {
	Type       string           `json:"type"`
	Group      string           `json:"group"`
	Messages   []KratosMessage  `json:"messages"`
	Meta       KratosNodeMeta   `json:"meta"`
	Attributes KratosAttributes `json:"attributes"`
}

type KratosNodeMeta struct {
	Label *struct {
		Text string `json:"text"`
	} `json:"label"`
}

type KratosAttributes struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
	ID    string `json:"id"`
	Src   string `json:"src"`
	Text  *struct {
		ID      int    `json:"id"`
		Text    string `json:"text"`
		Type    string `json:"type"`
		Context struct {
			Secret string `json:"secret"`
		} `json:"context"`
	} `json:"text"`
	Required bool `json:"required"`
	Disabled bool `json:"disabled"`
}

type KratosMessage struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type KratosFlowClient struct {
	apiBaseURL     string
	browserBaseURL string
	httpClient     *http.Client
}

func NewKratosFlowClient(apiBaseURL, browserBaseURL string) *KratosFlowClient {
	return &KratosFlowClient{
		apiBaseURL:     normalizeBaseURL(apiBaseURL),
		browserBaseURL: normalizeBaseURL(browserBaseURL),
		httpClient:     &http.Client{},
	}
}

func (c *KratosFlowClient) BrowserInitURL(flowType string) string {
	return c.browserBaseURL + "/self-service/" + flowType + "/browser"
}

func (c *KratosFlowClient) GetFlow(ctx context.Context, r *http.Request, flowType, flowID string) (KratosFlow, error) {
	endpoint := c.apiBaseURL + "/self-service/" + flowType + "/flows?" + url.Values{"id": []string{flowID}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return KratosFlow{}, fmt.Errorf("build kratos flow request: %w", err)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return KratosFlow{}, fmt.Errorf("call kratos flow: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return KratosFlow{}, fmt.Errorf("kratos flow returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var flow KratosFlow
	if err := json.NewDecoder(resp.Body).Decode(&flow); err != nil {
		return KratosFlow{}, fmt.Errorf("decode kratos flow: %w", err)
	}
	return flow, nil
}

type PageData struct {
	Title       string
	Description string
	FlowType    string
	Flow        KratosFlow
}

func RenderPage(w http.ResponseWriter, data PageData) error {
	const tmpl = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }} — Idol Auth</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: #0a0c12;
      background-image: radial-gradient(ellipse at 20% 50%, rgba(108,99,255,0.08) 0%, transparent 60%),
                        radial-gradient(ellipse at 80% 20%, rgba(99,179,237,0.05) 0%, transparent 50%);
      color: #e8eaf0;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }
    .card {
      width: 100%;
      max-width: 420px;
      background: #13161f;
      border: 1px solid rgba(255,255,255,0.08);
      border-radius: 16px;
      padding: 40px;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 32px;
    }
    .brand-icon {
      width: 32px;
      height: 32px;
      background: #6c63ff;
      border-radius: 7px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 15px;
      color: white;
    }
    .brand-name {
      font-size: 13px;
      font-weight: 600;
      color: #7c8394;
      letter-spacing: 0.1em;
      text-transform: uppercase;
    }
    h1 { margin: 0 0 8px; font-size: 24px; font-weight: 700; letter-spacing: -0.02em; }
    .description { color: #7c8394; margin: 0 0 28px; font-size: 14px; line-height: 1.6; }
    .alert {
      padding: 11px 14px;
      border-radius: 8px;
      margin-bottom: 14px;
      font-size: 13px;
      line-height: 1.5;
    }
    .alert-error {
      background: rgba(239,68,68,0.1);
      border: 1px solid rgba(239,68,68,0.25);
      color: #fca5a5;
    }
    .alert-info {
      background: rgba(99,179,237,0.1);
      border: 1px solid rgba(99,179,237,0.2);
      color: #93c5fd;
    }
    form { display: flex; flex-direction: column; gap: 18px; }
    .field { display: flex; flex-direction: column; gap: 6px; }
    label {
      font-size: 12px;
      font-weight: 600;
      color: #7c8394;
      letter-spacing: 0.06em;
      text-transform: uppercase;
    }
    input:not([type=hidden]):not([type=submit]) {
      background: rgba(255,255,255,0.04);
      border: 1px solid rgba(255,255,255,0.09);
      border-radius: 8px;
      color: #e8eaf0;
      font-size: 15px;
      padding: 11px 13px;
      outline: none;
      transition: border-color 0.15s, box-shadow 0.15s;
      width: 100%;
    }
    input:not([type=hidden]):not([type=submit]):focus {
      border-color: rgba(108,99,255,0.55);
      box-shadow: 0 0 0 3px rgba(108,99,255,0.12);
    }
    input[readonly] { opacity: 0.55; cursor: default; }
    button, input[type=submit] {
      background: #6c63ff;
      color: white;
      border: none;
      border-radius: 8px;
      font-size: 14px;
      font-weight: 600;
      padding: 12px;
      cursor: pointer;
      transition: opacity 0.15s, transform 0.1s;
      width: 100%;
      letter-spacing: 0.01em;
    }
    button:hover, input[type=submit]:hover { opacity: 0.88; }
    button:active, input[type=submit]:active { transform: scale(0.99); }
    .qr-wrap img {
      border-radius: 10px;
      background: white;
      padding: 12px;
      max-width: 200px;
    }
    .nav {
      display: flex;
      flex-wrap: wrap;
      gap: 4px 0;
      margin-top: 28px;
      padding-top: 22px;
      border-top: 1px solid rgba(255,255,255,0.07);
    }
    .nav a {
      font-size: 13px;
      color: #7c8394;
      text-decoration: none;
      padding: 2px 0;
      transition: color 0.15s;
    }
    .nav a:hover { color: #e8eaf0; }
    .nav a:not(:last-child)::after { content: '·'; margin: 0 10px; color: rgba(255,255,255,0.15); }
  </style>
</head>
<body>
  <div class="card">
    <div class="brand">
      <div class="brand-icon">✦</div>
      <span class="brand-name">Idol Auth</span>
    </div>
    <h1>{{ .Title }}</h1>
    <p class="description">{{ .Description }}</p>
    {{ range .Flow.Messages }}
      <div class="alert {{ if eq .Type "error" }}alert-error{{ else }}alert-info{{ end }}">{{ .Text }}</div>
    {{ end }}
    <form action="{{ .Flow.UI.Action }}" method="{{ .Flow.UI.Method }}">
      {{ range .Flow.UI.Nodes }}
        {{ range .Messages }}<div class="alert alert-error">{{ .Text }}</div>{{ end }}
        {{ if eq .Type "img" }}
          <div class="field qr-wrap">
            <label>{{ nodeLabel . }}</label>
            <img src="{{ imageSrc . }}" alt="{{ nodeLabel . }}">
          </div>
        {{ else if eq .Type "text" }}
          <div class="field">
            <label>{{ nodeLabel . }}</label>
            <input type="text" value="{{ textValue . }}" readonly>
          </div>
        {{ else if eq .Attributes.Type "hidden" }}
          <input type="hidden" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}">
        {{ else if eq .Attributes.Type "submit" }}
          <button type="submit" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}">{{ nodeLabel . }}</button>
        {{ else }}
          <div class="field">
            <label for="{{ .Attributes.Name }}">{{ nodeLabel . }}</label>
            <input id="{{ .Attributes.Name }}" name="{{ .Attributes.Name }}" type="{{ inputType .Attributes.Type }}" value="{{ .Attributes.Value }}" {{ if .Attributes.Required }}required{{ end }} {{ if .Attributes.Disabled }}disabled{{ end }}>
          </div>
        {{ end }}
      {{ end }}
    </form>
    <nav class="nav">
      <a href="/">ホーム</a>
      <a href="/login">ログイン</a>
      <a href="/registration">新規登録</a>
      <a href="/recovery">復旧</a>
      <a href="/verification">確認</a>
      <a href="/settings">設定</a>
    </nav>
  </div>
</body>
</html>`

	t := template.Must(template.New("page").Funcs(template.FuncMap{
		"inputType": func(inputType string) string {
			if inputType == "submit" || inputType == "hidden" {
				return inputType
			}
			if inputType == "" {
				return "text"
			}
			return inputType
		},
		"nodeLabel": func(node KratosNode) string {
			if node.Meta.Label != nil && node.Meta.Label.Text != "" {
				return node.Meta.Label.Text
			}
			if node.Attributes.Text != nil && node.Attributes.Text.Text != "" {
				return node.Attributes.Text.Text
			}
			if node.Attributes.Value != nil {
				return fmt.Sprint(node.Attributes.Value)
			}
			return node.Attributes.Name
		},
		"imageSrc": func(node KratosNode) template.URL {
			return template.URL(node.Attributes.Src)
		},
		"textValue": func(node KratosNode) string {
			if node.Attributes.Text != nil {
				if node.Attributes.Text.Context.Secret != "" {
					return node.Attributes.Text.Context.Secret
				}
				return node.Attributes.Text.Text
			}
			if node.Attributes.Value != nil {
				return fmt.Sprint(node.Attributes.Value)
			}
			return ""
		},
	}).Parse(tmpl))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.Execute(w, data)
}
