package notification

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/ca-x/vaultwarden-syncer/internal/config"
)

type Service struct {
	config *config.NotificationConfig
}

func NewService(config *config.NotificationConfig) *Service {
	return &Service{
		config: config,
	}
}

// SendFailureNotification 发送失败通知
func (s *Service) SendFailureNotification(subject, message string) error {
	if !s.config.Email.Enabled {
		return nil
	}

	emailConfig := s.config.Email

	// 构建邮件内容
	msg := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"\r\n"+
			"%s",
		emailConfig.From,
		emailConfig.To,
		subject,
		message,
	)

	// 发送邮件
	auth := smtp.PlainAuth("", emailConfig.Username, emailConfig.Password, emailConfig.SMTPHost)
	addr := fmt.Sprintf("%s:%d", emailConfig.SMTPHost, emailConfig.SMTPPort)

	err := smtp.SendMail(addr, auth, emailConfig.From, strings.Split(emailConfig.To, ","), []byte(msg))
	if err != nil {
		log.Printf("Failed to send email notification: %v", err)
		return err
	}

	log.Printf("Failure notification sent successfully to %s", emailConfig.To)
	return nil
}

// SendHealthCheckReport 发送健康检查报告
func (s *Service) SendHealthCheckReport(results map[string]error) error {
	if !s.config.Email.Enabled {
		return nil
	}

	var failed []string
	var passed []string

	for storage, err := range results {
		if err != nil {
			failed = append(failed, fmt.Sprintf("- %s: %v", storage, err))
		} else {
			passed = append(passed, fmt.Sprintf("- %s: OK", storage))
		}
	}

	var message strings.Builder
	if len(failed) > 0 {
		message.WriteString("Failed storage backends:\r\n")
		message.WriteString(strings.Join(failed, "\r\n"))
		message.WriteString("\r\n\r\n")
	}

	if len(passed) > 0 {
		message.WriteString("Healthy storage backends:\r\n")
		message.WriteString(strings.Join(passed, "\r\n"))
		message.WriteString("\r\n")
	}

	subject := fmt.Sprintf("Vaultwarden Sync Health Check Report (%d failed, %d passed)",
		len(failed), len(passed))

	return s.SendFailureNotification(subject, message.String())
}
