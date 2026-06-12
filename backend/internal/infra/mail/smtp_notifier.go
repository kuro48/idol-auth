// Package mail provides email notifications for the appreg domain.
package mail

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	netmail "net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"text/template"

	"github.com/kuro48/idol-auth/internal/config"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

// SMTPNotifier sends transactional emails via SMTP.
type SMTPNotifier struct {
	from        string
	appName     string
	baseURL     string
	addr        string
	auth        smtp.Auth
	adminEmails []string
}

// NewSMTPNotifier builds a notifier from MailConfig. adminEmails receive a
// copy whenever a new registration is submitted, so operators keep visibility
// over self-service (auto-approved) registrations without a review gate.
// Returns NoopNotifier when mail is disabled or SMTP_URL is empty.
func NewSMTPNotifier(cfg config.MailConfig, adminEmails []string) appreg.Notifier {
	if !cfg.Enabled || strings.TrimSpace(cfg.SMTPURL) == "" {
		return appreg.NoopNotifier{}
	}
	u, err := url.Parse(cfg.SMTPURL)
	if err != nil {
		return appreg.NoopNotifier{}
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "587"
	}

	var auth smtp.Auth
	if u.User != nil {
		pass, _ := u.User.Password()
		auth = smtp.PlainAuth("", u.User.Username(), pass, host)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &SMTPNotifier{
		from:        cfg.From,
		appName:     cfg.AppName,
		baseURL:     baseURL,
		addr:        host + ":" + port,
		auth:        auth,
		adminEmails: adminEmails,
	}
}

func (n *SMTPNotifier) NotifySubmitted(_ context.Context, req appreg.Request) error {
	subject := fmt.Sprintf("[%s] Your app registration request has been received", n.appName)
	body := n.render(submittedTmpl, map[string]any{
		"AppName":      n.appName,
		"RequestName":  req.Name,
		"DashboardURL": n.baseURL + "/developer/applications/" + req.ID.String(),
	})
	err := n.send(req.ContactEmail, subject, body)
	n.notifyAdmins(req)
	return err
}

// notifyAdmins sends operators a copy for every new registration. Failures are
// non-fatal: admin visibility must never block a developer's registration.
func (n *SMTPNotifier) notifyAdmins(req appreg.Request) {
	if len(n.adminEmails) == 0 {
		return
	}
	subject := fmt.Sprintf("[%s] New app registration: %s", n.appName, req.Name)
	body := n.render(adminSubmittedTmpl, map[string]any{
		"AppName":      n.appName,
		"RequestName":  req.Name,
		"ContactEmail": req.ContactEmail,
		"AdminURL":     n.baseURL + "/admin/app-requests/" + req.ID.String(),
	})
	for _, adminEmail := range n.adminEmails {
		_ = n.send(adminEmail, subject, body)
	}
}

func (n *SMTPNotifier) NotifyApproved(_ context.Context, req appreg.Request) error {
	subject := fmt.Sprintf("[%s] Your app registration request has been approved", n.appName)
	body := n.render(approvedTmpl, map[string]any{
		"AppName":      n.appName,
		"RequestName":  req.Name,
		"DashboardURL": n.baseURL + "/developer/applications/" + req.ID.String(),
	})
	return n.send(req.ContactEmail, subject, body)
}

func (n *SMTPNotifier) NotifyRejected(_ context.Context, req appreg.Request, reason string) error {
	subject := fmt.Sprintf("[%s] Your app registration request has been rejected", n.appName)
	body := n.render(rejectedTmpl, map[string]any{
		"AppName":      n.appName,
		"RequestName":  req.Name,
		"Reason":       reason,
		"DashboardURL": n.baseURL + "/developer/applications/" + req.ID.String(),
	})
	return n.send(req.ContactEmail, subject, body)
}

func (n *SMTPNotifier) NotifyChangesRequested(_ context.Context, req appreg.Request, reason string) error {
	subject := fmt.Sprintf("[%s] Changes requested for your app registration", n.appName)
	body := n.render(changesRequestedTmpl, map[string]any{
		"AppName":      n.appName,
		"RequestName":  req.Name,
		"Reason":       reason,
		"DashboardURL": n.baseURL + "/developer/applications/" + req.ID.String(),
	})
	return n.send(req.ContactEmail, subject, body)
}

func (n *SMTPNotifier) send(to, subject, body string) error {
	msg, envelopeFrom, envelopeTo, err := buildMessage(n.from, to, subject, body)
	if err != nil {
		return err
	}
	return smtp.SendMail(n.addr, n.auth, envelopeFrom, []string{envelopeTo}, []byte(msg))
}

func buildMessage(from, to, subject, body string) (string, string, string, error) {
	fromAddr, err := netmail.ParseAddress(from)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid from address: %w", err)
	}
	toAddr, err := netmail.ParseAddress(to)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid to address: %w", err)
	}
	if containsHeaderNewline(subject) {
		return "", "", "", fmt.Errorf("invalid subject header")
	}

	var sb strings.Builder
	sb.WriteString("From: " + fromAddr.String() + "\r\n")
	sb.WriteString("To: " + toAddr.String() + "\r\n")
	sb.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return sb.String(), fromAddr.Address, toAddr.Address, nil
}

func containsHeaderNewline(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}

func (n *SMTPNotifier) render(tmpl *template.Template, data map[string]any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("(template render error: %s)", err)
	}
	return buf.String()
}

var submittedTmpl = template.Must(template.New("submitted").Parse(`Hello,

Thank you for submitting your application registration request for "{{.RequestName}}" on {{.AppName}}.

Your request is now under review. We will notify you once a decision has been made.

You can check the status of your request here:
{{.DashboardURL}}

— The {{.AppName}} Team
`))

var adminSubmittedTmpl = template.Must(template.New("admin_submitted").Parse(`A new app registration request has been submitted on {{.AppName}}.

App name: {{.RequestName}}
Contact: {{.ContactEmail}}

Review it here:
{{.AdminURL}}
`))

var approvedTmpl = template.Must(template.New("approved").Parse(`Hello,

Great news! Your application registration request for "{{.RequestName}}" has been approved.

Visit your developer dashboard to retrieve your client credentials:
{{.DashboardURL}}

Important: your client_secret is displayed only once. Please copy and store it securely.

— The {{.AppName}} Team
`))

var rejectedTmpl = template.Must(template.New("rejected").Parse(`Hello,

We regret to inform you that your application registration request for "{{.RequestName}}" has been rejected.
{{if .Reason}}
Reason provided by the reviewer:
{{.Reason}}
{{end}}
You can view your request details here:
{{.DashboardURL}}

If you have questions, please contact support.

— The {{.AppName}} Team
`))

var changesRequestedTmpl = template.Must(template.New("changes").Parse(`Hello,

The reviewer has requested changes to your application registration request for "{{.RequestName}}".
{{if .Reason}}
Reviewer feedback:
{{.Reason}}
{{end}}
Please update and resubmit your request here:
{{.DashboardURL}}

— The {{.AppName}} Team
`))
