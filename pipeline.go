package mailg

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
)

func (c *Client) generateMessage(
	done <-chan interface{},
	seqset *imap.SeqSet,
	items []imap.FetchItem,
) <-chan *imap.Message {
	messageStream := make(chan *imap.Message)
	go func() {
		c.imapClient.Fetch(seqset, items, messageStream)
		for {
			select {
			case <-done:
				return
			}
		}
	}()
	return messageStream
}

func (c *Client) toMail(
	done <-chan interface{},
	messageStream <-chan *imap.Message,
	items *MailItems,
) <-chan *Mail {
	mailStream := make(chan *Mail)
	go func() {
		defer close(mailStream)
		for m := range messageStream {
			select {
			case <-done:
				return
			case mailStream <- c.messsageToMail(m, items):
			}
		}
	}()
	return mailStream
}

func (c *Client) messsageToMail(m *imap.Message, items *MailItems) *Mail {
	nm := new(Mail)

	r, err := mail.CreateReader(m.GetBody(c.section))
	if err != nil {
		return nil
	}

	h := r.Header

	if items.has(date) {
		nm.Date, err = h.Date()
		if err != nil {
			return nil
		}
	}

	if items.has(from) {
		nm.From, err = addressToStr(h.AddressList("From"))
		if err != nil {
			return nil
		}
	}

	if items.has(to) {
		nm.To, err = addressToStr(h.AddressList("To"))
		if err != nil {
			return nil
		}
	}

	if items.has(cc) {
		nm.Cc, err = addressToStr(h.AddressList("Cc"))
		if err != nil {
			return nil
		}
	}

	if items.has(subject) {
		nm.Subject, err = h.Subject()
		if err != nil {
			return nil
		}
	}

	if items.hasTextORAttachment() {
		for {
			p, err := r.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil
			}

			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				if items.has(text) {
					b, err := ioutil.ReadAll(p.Body)
					if err != nil {
						return nil
					}
					nm.Text = string(b)
				}
			case *mail.AttachmentHeader:
				if items.has(attachment) {
					fileName, err := h.Filename()
					if err != nil {
						return nil
					}

					buf := bytes.NewBuffer(nil)
					_, err = buf.ReadFrom(p.Body)
					if err != nil {
						return nil
					}
					nm.Attachments = append(nm.Attachments, &Attachment{Filename: fileName, Reader: buf})
				}
			}
		}
	}
	return nm
}

func addressToStr(as []*mail.Address, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	var s []string
	for _, a := range as {
		s = append(s, a.Address)
	}
	return s, nil
}

func toAttachment(
	done <-chan interface{},
	mailStream <-chan *Mail,
) <-chan *Attachment {
	attachmentStream := make(chan *Attachment)
	go func() {
		defer close(attachmentStream)
		for m := range mailStream {
			for _, a := range m.Attachments {
				select {
				case <-done:
					return
				case attachmentStream <- a:
				}
			}
		}
	}()
	return attachmentStream
}
