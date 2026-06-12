package mail

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/kuro48/idol-auth/internal/config"
)

// AccountEmailResolver resolves an identity ID to an email address.
type AccountEmailResolver interface {
	GetIdentityEmail(ctx context.Context, identityID string) (string, error)
}

// AccountSMTPNotifier sends account lifecycle emails via SMTP.
type AccountSMTPNotifier struct {
	from     string
	appName  string
	baseURL  string
	addr     string
	resolver AccountEmailResolver
}

// NewAccountSMTPNotifier builds an AccountSMTPNotifier from MailConfig.
// Returns nil when mail is disabled or SMTP_URL is empty, so callers can
// guard with a nil check (the WithAccountMailer option accepts nil).
func NewAccountSMTPNotifier(cfg config.MailConfig, resolver AccountEmailResolver) *AccountSMTPNotifier {
	if !cfg.Enabled || strings.TrimSpace(cfg.SMTPURL) == "" {
		return nil
	}
	u, err := url.Parse(cfg.SMTPURL)
	if err != nil {
		return nil
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "587"
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &AccountSMTPNotifier{
		from:     cfg.From,
		appName:  cfg.AppName,
		baseURL:  baseURL,
		addr:     host + ":" + port,
		resolver: resolver,
	}
}

func (n *AccountSMTPNotifier) NotifyDeletionScheduled(ctx context.Context, identityID string, scheduledFor time.Time) error {
	email, err := n.resolver.GetIdentityEmail(ctx, identityID)
	if err != nil {
		return fmt.Errorf("account notifier: resolve email: %w", err)
	}
	if strings.TrimSpace(email) == "" {
		return nil
	}
	subject := fmt.Sprintf("[%s] Your account is scheduled for deletion", n.appName)
	body := execTemplate(deletionScheduledTmpl, map[string]any{
		"AppName":      n.appName,
		"ScheduledFor": scheduledFor.UTC().Format("2006-01-02"),
		"AccountURL":   n.baseURL + "/account",
	})
	helper := &SMTPNotifier{from: n.from, addr: n.addr}
	return helper.send(email, subject, body)
}

func execTemplate(tmpl *template.Template, data map[string]any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("(template render error: %s)", err)
	}
	return buf.String()
}

var deletionScheduledTmpl = template.Must(template.New("deletion_scheduled").Parse(`Hello,

Your account on {{.AppName}} has been scheduled for deletion on {{.ScheduledFor}}.

If you did not request this, you can cancel the deletion by signing in to your account before that date:
{{.AccountURL}}

If you want to proceed with deletion, no action is needed.

— The {{.AppName}} Team
`))
