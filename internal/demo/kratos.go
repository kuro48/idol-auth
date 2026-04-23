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
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
  <style>
    body { font-family: Georgia, serif; background: linear-gradient(135deg, #f7f1e8, #d7e5f0); color: #1f2933; margin: 0; }
    main { max-width: 720px; margin: 40px auto; background: rgba(255,255,255,0.88); padding: 32px; border-radius: 20px; box-shadow: 0 24px 60px rgba(31,41,51,0.12); }
    h1 { margin-top: 0; }
    form { display: grid; gap: 16px; }
    label { display: block; font-weight: 600; margin-bottom: 6px; }
    input, button { width: 100%; padding: 12px; border-radius: 10px; border: 1px solid #c7d2da; font-size: 16px; }
    button, input[type="submit"] { background: #1f6f8b; color: white; border: 0; cursor: pointer; }
    .message { padding: 12px; border-radius: 10px; background: #f9e2d2; margin-bottom: 12px; }
    .links { display: flex; gap: 12px; flex-wrap: wrap; margin-top: 24px; }
    .links a { color: #1f6f8b; }
  </style>
</head>
<body>
  <main>
    <h1>{{ .Title }}</h1>
    <p>{{ .Description }}</p>
    {{ range .Flow.Messages }}<div class="message">{{ .Text }}</div>{{ end }}
    <form action="{{ .Flow.UI.Action }}" method="{{ .Flow.UI.Method }}">
      {{ range .Flow.UI.Nodes }}
        {{ range .Messages }}<div class="message">{{ .Text }}</div>{{ end }}
        {{ if eq .Type "img" }}
          <div>
            <label>{{ nodeLabel . }}</label>
            <img src="{{ imageSrc . }}" alt="{{ nodeLabel . }}" style="max-width: 256px; border-radius: 12px; border: 1px solid #c7d2da; background: white; padding: 12px;">
          </div>
        {{ else if eq .Type "text" }}
          <div>
            <label>{{ nodeLabel . }}</label>
            <input type="text" value="{{ textValue . }}" readonly>
          </div>
        {{ else if eq .Attributes.Type "hidden" }}
          <input type="hidden" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}">
        {{ else if eq .Attributes.Type "submit" }}
          <button type="submit" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}">{{ nodeLabel . }}</button>
        {{ else }}
          <div>
            <label for="{{ .Attributes.Name }}">{{ nodeLabel . }}</label>
            <input id="{{ .Attributes.Name }}" name="{{ .Attributes.Name }}" type="{{ inputType .Attributes.Type }}" value="{{ .Attributes.Value }}" {{ if .Attributes.Required }}required{{ end }} {{ if .Attributes.Disabled }}disabled{{ end }}>
          </div>
        {{ end }}
      {{ end }}
    </form>
    <div class="links">
      <a href="/">Home</a>
      <a href="/login">Login</a>
      <a href="/registration">Registration</a>
      <a href="/recovery">Recovery</a>
      <a href="/verification">Verification</a>
      <a href="/settings">Settings</a>
    </div>
  </main>
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
