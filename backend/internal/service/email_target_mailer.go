package service

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"mime"
	"strings"
	"time"

	"linuxdospace/backend/internal/config"
	"linuxdospace/backend/internal/mailrelay"
)

const (
	// emailTargetVerificationTokenLifetime keeps verification links short-lived
	// so stale leaked links cannot be replayed indefinitely.
	emailTargetVerificationTokenLifetime = 24 * time.Hour

	// emailTargetVerificationSendTimeout bounds one outbound verification-email
	// attempt so the binding API cannot hang forever on a slow remote MX host.
	emailTargetVerificationSendTimeout = 45 * time.Second
)

// EmailTargetVerificationMailInput describes the exact email the backend sends
// to prove that the current user really controls the claimed target inbox.
type EmailTargetVerificationMailInput struct {
	TargetEmail      string
	VerificationURL  string
	ExpiresAt        time.Time
	AppDisplayName   string
	ForwardFrom      string
	FrontendEmailURL string
}

// EmailTargetVerificationMailer abstracts the outbound delivery of one target
// verification email so tests can replace the real SMTP implementation.
type EmailTargetVerificationMailer interface {
	SendVerification(ctx context.Context, input EmailTargetVerificationMailInput) error
}

// directMXEmailTargetVerificationMailer sends the verification email through
// the same direct-MX SMTP forwarder the production relay already uses.
type directMXEmailTargetVerificationMailer struct {
	forwarder mailrelay.MessageForwarder
	now       func() time.Time
}

// newEmailTargetVerificationMailer constructs the production mailer used by
// the target-binding flow. The returned value still validates its inputs at
// send time so tests can pass minimal configs without crashing constructors.
func newEmailTargetVerificationMailer(cfg config.Config) EmailTargetVerificationMailer {
	return &directMXEmailTargetVerificationMailer{
		forwarder: mailrelay.NewSMTPForwarder(cfg.Mail),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// SendVerification delivers one plain-text verification email directly to the
// claimed target inbox.
func (m *directMXEmailTargetVerificationMailer) SendVerification(ctx context.Context, input EmailTargetVerificationMailInput) error {
	if m == nil || m.forwarder == nil {
		return fmt.Errorf("email target verification mailer is not configured")
	}

	targetEmail := strings.ToLower(strings.TrimSpace(input.TargetEmail))
	if targetEmail == "" {
		return fmt.Errorf("verification target email is empty")
	}
	if _, err := mail.ParseAddress(targetEmail); err != nil {
		return fmt.Errorf("verification target email is invalid: %w", err)
	}

	forwardFrom := strings.TrimSpace(input.ForwardFrom)
	if forwardFrom == "" {
		return fmt.Errorf("verification sender address is empty")
	}
	if _, err := mail.ParseAddress(forwardFrom); err != nil {
		return fmt.Errorf("verification sender address is invalid: %w", err)
	}

	verificationURL := strings.TrimSpace(input.VerificationURL)
	if verificationURL == "" {
		return fmt.Errorf("verification url is empty")
	}
	if _, err := url.ParseRequestURI(verificationURL); err != nil {
		return fmt.Errorf("verification url is invalid: %w", err)
	}

	rawMessage, err := buildEmailTargetVerificationMessage(m.now(), input)
	if err != nil {
		return err
	}

	sendCtx, cancel := context.WithTimeout(ctx, emailTargetVerificationSendTimeout)
	defer cancel()

	return m.forwarder.Forward(sendCtx, mailrelay.ForwardRequest{
		OriginalEnvelopeFrom: forwardFrom,
		OriginalEnvelopeTo:   []string{targetEmail},
		TargetRecipients:     []string{targetEmail},
		RawMessage:           rawMessage,
	})
}

// buildEmailTargetVerificationMessage renders the RFC 5322 message sent to one
// claimed target inbox. The content stays intentionally plain so common mail
// providers can render it without HTML sanitization surprises.
func buildEmailTargetVerificationMessage(now time.Time, input EmailTargetVerificationMailInput) ([]byte, error) {
	appDisplayName := strings.TrimSpace(input.AppDisplayName)
	if appDisplayName == "" {
		appDisplayName = "LinuxDoSpace"
	}

	frontendEmailURL := strings.TrimSpace(input.FrontendEmailURL)
	if frontendEmailURL == "" {
		frontendEmailURL = "/emails"
	}

	message := strings.Join([]string{
		fmt.Sprintf("From: %s", input.ForwardFrom),
		fmt.Sprintf("To: %s", input.TargetEmail),
		fmt.Sprintf("Subject: %s", mime.BEncoding.Encode("utf-8", appDisplayName+" 目标邮箱验证")),
		fmt.Sprintf("Date: %s", now.Format(time.RFC1123Z)),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		"你好，",
		"",
		fmt.Sprintf("你正在将 %s 绑定为 %s 的转发目标邮箱。", input.TargetEmail, appDisplayName),
		"请打开下面的链接完成验证：",
		input.VerificationURL,
		"",
		fmt.Sprintf("该验证链接将在 %s 失效。", input.ExpiresAt.UTC().Format("2006-01-02 15:04:05 UTC")),
		"",
		"如果这不是你本人发起的操作，请直接忽略这封邮件。",
		fmt.Sprintf("验证完成后，可以回到 %s 继续配置邮箱转发。", frontendEmailURL),
		"",
	}, "\r\n")

	return []byte(message), nil
}
