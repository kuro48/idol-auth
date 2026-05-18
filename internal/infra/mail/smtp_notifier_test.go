package mail

import (
	"strings"
	"testing"
)

func TestBuildMessageRejectsHeaderInjection(t *testing.T) {
	_, _, _, err := buildMessage("Idol Auth <noreply@example.com>", "dev@example.com", "hello\r\nBcc: attacker@example.com", "body")
	if err == nil {
		t.Fatal("expected header injection to be rejected")
	}
}

func TestBuildMessageFormatsAddressesAndSubject(t *testing.T) {
	msg, envelopeFrom, envelopeTo, err := buildMessage("Idol Auth <noreply@example.com>", "Dev <dev@example.com>", "承認されました", "body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelopeFrom != "noreply@example.com" || envelopeTo != "dev@example.com" {
		t.Fatalf("unexpected envelope addresses: from=%q to=%q", envelopeFrom, envelopeTo)
	}
	if !strings.Contains(msg, "From: \"Idol Auth\" <noreply@example.com>\r\n") {
		t.Fatalf("expected formatted From header, got %q", msg)
	}
	if !strings.Contains(msg, "Subject: =?utf-8?") {
		t.Fatalf("expected encoded subject, got %q", msg)
	}
}
