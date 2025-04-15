package service

import (
	"context"
	"log"
	"time"

	"gopkg.in/gomail.v2"
)

type EmailClient struct {
	dialer *gomail.Dialer
	logger *log.Logger
	from   string
}

func NewEmailClient(host string, port int, username, password, from string, logger *log.Logger) *EmailClient {
	dialer := gomail.NewDialer(host, port, username, password)
	return &EmailClient{
		dialer: dialer,
		logger: logger,
		from:   from,
	}
}

func (c *EmailClient) SendVerificationEmail(ctx context.Context, toEmail, userID, token string) error {
	activationLink := "http://example.com/api/auth/verify?user_id=" + userID + "&token=" + token
	m := gomail.NewMessage()
	m.SetHeader("From", c.from)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "ACTIVATE ACCOUNT")
	m.SetBody("text/plain", "Please activate your account by clicking the link: "+activationLink)
	m.AddAlternative("text/html", "<p>Please activate your account by clicking the link: <a href=\""+activationLink+"\">Activate</a></p>")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		c.logger.Printf("Email send to %s cancelled: %v", toEmail, ctx.Err())
		return ctx.Err()
	default:
		if err := c.dialer.DialAndSend(m); err != nil {
			c.logger.Printf("Failed to send email to %s: %v", toEmail, err)
			return err
		}
		c.logger.Printf("Verification email sent to %s", toEmail)
		return nil
	}
}
