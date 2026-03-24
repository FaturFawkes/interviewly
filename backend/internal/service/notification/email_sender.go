package notification

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/logger"
	"go.uber.org/zap"
)

type otpEmailSender interface {
	SendRegistrationOTP(email, otp string) error
}

type smtpOTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
}

type logOTPSender struct{}

func NewRegistrationOTPSender(cfg *config.Config) otpEmailSender {
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		return &logOTPSender{}
	}

	from := strings.TrimSpace(cfg.SMTPFromEmail)
	if from == "" {
		from = "no-reply@interviewly.local"
	}

	return &smtpOTPSender{
		host:     strings.TrimSpace(cfg.SMTPHost),
		port:     cfg.SMTPPort,
		username: strings.TrimSpace(cfg.SMTPUsername),
		password: cfg.SMTPPassword,
		from:     from,
		fromName: strings.TrimSpace(cfg.SMTPFromName),
	}
}

func (s *smtpOTPSender) SendRegistrationOTP(email, otp string) error {
	address := fmt.Sprintf("%s:%d", s.host, s.port)
	fromHeader := s.from
	if s.fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", s.fromName, s.from)
	}

	subject := "Interviewly - Registration OTP"
	body := fmt.Sprintf("Your OTP code is: %s\n\nThis code will expire in 10 minutes.", otp)
	message := []byte(
		fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", fromHeader, email, subject, body),
	)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	return smtp.SendMail(address, auth, s.from, []string{email}, message)
}

func (s *logOTPSender) SendRegistrationOTP(email, otp string) error {
	logger.L().Info("[notification] OTP delivery fallback (SMTP not configured)", zap.String("email", email), zap.String("otp", otp))
	return nil
}
