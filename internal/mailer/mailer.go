package mailer

import (
	"bytes"
	"crypto/tls"
	"embed"
	"fmt"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
	dialer *mail.Dialer
	sender string
}

func New(host string, port int, username, password, sender string) Mailer {
	// Initialize a new mail.Dialer instance with the given SMTP server settings.
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 10 * time.Second // Increased timeout for better reliability
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

func (m Mailer) Send(recipient, templateFile string, data interface{}) error {
	// Parse the template file from the embedded file system.
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return fmt.Errorf("error parsing template %s: %w", templateFile, err)
	}

	// Execute templates for subject, plain text body, and HTML body.
	subject, err := executeTemplate(tmpl, "subject", data)
	if err != nil {
		return fmt.Errorf("error executing subject template: %w", err)
	}

	plainBody, err := executeTemplate(tmpl, "plainBody", data)
	if err != nil {
		return fmt.Errorf("error executing plainBody template: %w", err)
	}

	htmlBody, err := executeTemplate(tmpl, "htmlBody", data)
	if err != nil {
		return fmt.Errorf("error executing htmlBody template: %w", err)
	}

	// Create a new message and set headers and body.
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", plainBody)
	msg.AddAlternative("text/html", htmlBody)

	// Attempt to send the email up to three times.
	for i := 0; i < 3; i++ {
		err = m.dialer.DialAndSend(msg)
		if err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond) // Short delay between attempts
	}

	return fmt.Errorf("failed to send email after 3 attempts to %s: %w", recipient, err)
}

// executeTemplate executes the specified template and returns the result as a string.
func executeTemplate(tmpl *template.Template, name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, name, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
