package bridge

import (
	"github.com/bitcode-framework/bitcode/pkg/email"
)

type emailBridge struct {
	sender email.Sender
}

func newEmailBridge(sender email.Sender) *emailBridge {
	return &emailBridge{sender: sender}
}

func (e *emailBridge) Send(opts EmailOptions) error {
	if !e.sender.IsConfigured() {
		return NewError(ErrEmailNotConfigured, "email not configured")
	}

	body := opts.Body
	// TODO: template rendering support (opts.Template + opts.Data)

	return e.sender.Send(opts.To, opts.Subject, body)
}
