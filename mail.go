package goattach

import (
	"bytes"
	"io"
	"io/ioutil"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
)

type Mail struct {
	Date        time.Time
	From        string
	To          string
	Cc          string
	Sub         string
	Text        string
	Attachments []*Attachment
}

type Attachment struct {
	Filename string
	io.Reader
}

func newChanPipeMail(buffer int) (chan *Mail, chan error) {
	return make(chan *Mail, buffer), make(chan error, 1)
}

func pipeMail(dst chan *Mail, src chan *imap.Message, section *imap.BodySectionName, items ...MailItem) error {
	// src は送信元で close してくれる
	defer close(dst)
	for m := range src {
		nm := new(Mail)

		r, err := mail.CreateReader(m.GetBody(section))
		if err != nil {
			return err
		}

		h := r.Header

		if hasItem(items, Date) {
			nm.Date, err = h.Date()
			if err != nil {
				return err
			}
		}

		if hasItem(items, From) {
			nm.From, err = h.Text("From")
			if err != nil {
				return err
			}
		}

		if hasItem(items, To) {
			nm.To, err = h.Text("To")
			if err != nil {
				return err
			}
		}

		if hasItem(items, Cc) {
			nm.Cc, err = h.Text("Cc")
			if err != nil {
				return err
			}
		}

		if hasItem(items, Sub) {
			nm.Sub, err = h.Subject()
			if err != nil {
				return err
			}
		}

		if hasTextORAttachments(items) {
			for {
				p, err := r.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				switch h := p.Header.(type) {
				case *mail.InlineHeader:
					if hasItem(items, Text) {
						b, err := ioutil.ReadAll(p.Body)
						if err != nil {
							return err
						}
						nm.Text = string(b)
					}
				case *mail.AttachmentHeader:
					if hasItem(items, Attachments) {
						fileName, err := h.Filename()
						if err != nil {
							return err
						}

						buf := new(bytes.Buffer)
						buf.ReadFrom(utilPipe(p.Body))
						nm.Attachments = append(nm.Attachments, &Attachment{
							Filename: fileName,
							Reader:   utilPipe(buf),
						})
					}
				}
			}
		}
		dst <- nm
	}
	return nil
}

func utilPipe(src io.Reader) *io.PipeReader {
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		io.Copy(w, src)
	}()
	return r
}

func copyPipe(src io.Reader) *io.PipeReader {
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		buf := new(bytes.Buffer)
		io.Copy(buf, src)
		// buf.ReadFrom(src)
		io.Copy(w, buf)
	}()
	return r
}
