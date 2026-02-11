package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/redmonkez12/go-api-template/internal/logging"
)

type Service struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	fromEmail    string
	frontendURL  string
}

func NewService(smtpHost, smtpPort, smtpUser, smtpPassword, frontendURL string) *Service {
	return &Service{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
		fromEmail:    smtpUser,
		frontendURL:  frontendURL,
	}
}

// SendVerificationEmail sends an email verification link to the user
// This method is designed to be called in a goroutine
func (s *Service) SendVerificationEmail(ctx context.Context, toEmail, token string) error {
	logger := logging.GetLoggerFromContext(ctx)

	verificationLink := fmt.Sprintf("%s/verify?token=%s", s.frontendURL, token)

	subject := "Verify your email address"
	body, err := s.renderVerificationEmailTemplate(verificationLink)
	if err != nil {
		logger.Error("failed to render email template", "error", err)
		return fmt.Errorf("render template: %w", err)
	}

	if err := s.sendEmail(toEmail, subject, body); err != nil {
		logger.Error("failed to send verification email", "email", toEmail, "error", err)
		return fmt.Errorf("send email: %w", err)
	}

	logger.Info("verification email sent", "email", toEmail)
	return nil
}

// SendPasswordResetEmail sends a password reset link to the user
// This method is designed to be called in a goroutine
func (s *Service) SendPasswordResetEmail(ctx context.Context, toEmail, token string) error {
	logger := logging.GetLoggerFromContext(ctx)

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, token)

	subject := "Reset your password"
	body, err := s.renderPasswordResetEmailTemplate(resetLink)
	if err != nil {
		logger.Error("failed to render password reset email template", "error", err)
		return fmt.Errorf("render template: %w", err)
	}

	if err := s.sendEmail(toEmail, subject, body); err != nil {
		logger.Error("failed to send password reset email", "email", toEmail, "error", err)
		return fmt.Errorf("send email: %w", err)
	}

	logger.Info("password reset email sent", "email", toEmail)
	return nil
}

func (s *Service) sendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)

	// Build message
	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		s.fromEmail, to, subject, body,
	))

	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)
	return smtp.SendMail(addr, auth, s.fromEmail, []string{to}, msg)
}

func (s *Service) renderVerificationEmailTemplate(verificationLink string) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background-color: #4F46E5;
            color: white;
            padding: 20px;
            text-align: center;
            border-radius: 5px 5px 0 0;
        }
        .content {
            background-color: #f9f9f9;
            padding: 30px;
            border-radius: 0 0 5px 5px;
        }
        .button {
            display: inline-block;
            background-color: #4F46E5;
            color: white !important;
            padding: 12px 30px;
            text-decoration: none;
            border-radius: 5px;
            margin: 20px 0;
        }
        .footer {
            margin-top: 30px;
            font-size: 12px;
            color: #666;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Welcome!</h1>
    </div>
    <div class="content">
        <h2>Verify your email address</h2>
        <p>Thank you for signing up! Please click the button below to verify your email address and activate your account.</p>

        <a href="{{.VerificationLink}}" class="button" style="color: white !important;">Verify Email Address</a>

        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #4F46E5;">{{.VerificationLink}}</p>

        <p style="margin-top: 30px;">If you didn't create an account, you can safely ignore this email.</p>
    </div>
    <div class="footer">
        <p>This link will expire in 24 hours.</p>
        <p>&copy; 2026 Your App. All rights reserved.</p>
    </div>
</body>
</html>
`

	t, err := template.New("verification").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	data := struct {
		VerificationLink string
	}{
		VerificationLink: verificationLink,
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

func (s *Service) renderPasswordResetEmailTemplate(resetLink string) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background-color: #4F46E5;
            color: white;
            padding: 20px;
            text-align: center;
            border-radius: 5px 5px 0 0;
        }
        .content {
            background-color: #f9f9f9;
            padding: 30px;
            border-radius: 0 0 5px 5px;
        }
        .button {
            display: inline-block;
            background-color: #4F46E5;
            color: white !important;
            padding: 12px 30px;
            text-decoration: none;
            border-radius: 5px;
            margin: 20px 0;
        }
        .footer {
            margin-top: 30px;
            font-size: 12px;
            color: #666;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Password Reset Request</h1>
    </div>
    <div class="content">
        <h2>Reset your password</h2>
        <p>You requested to reset your password. Click the button below to create a new password.</p>

        <a href="{{.ResetLink}}" class="button" style="color: white !important;">Reset Password</a>

        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #4F46E5;">{{.ResetLink}}</p>

        <p style="margin-top: 30px;">If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>
    </div>
    <div class="footer">
        <p>This link will expire in 1 hour.</p>
        <p>&copy; 2026 Your App. All rights reserved.</p>
    </div>
</body>
</html>
`

	t, err := template.New("passwordReset").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	data := struct {
		ResetLink string
	}{
		ResetLink: resetLink,
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
