package mailg

import (
	"io"
	"time"
)

type Mail struct {
	Date        time.Time
	From        []string
	To          []string
	Cc          []string
	Subject     string
	Text        string
	Attachments []*Attachment
}

type Attachment struct {
	FileName string
	io.Reader
}

type mailItem int

const (
	date mailItem = iota
	from
	to
	cc
	subject
	text
	attachment
)

type MailItems []mailItem

func NewMailItems() *MailItems {
	return new(MailItems)
}

func (s *MailItems) Date() *MailItems {
	*s = append(*s, date)
	return s
}

func (s *MailItems) From() *MailItems {
	*s = append(*s, from)
	return s
}

func (s *MailItems) To() *MailItems {
	*s = append(*s, to)
	return s
}

func (s *MailItems) Cc() *MailItems {
	*s = append(*s, cc)
	return s
}

func (s *MailItems) Subject() *MailItems {
	*s = append(*s, subject)
	return s
}

func (s *MailItems) Text() *MailItems {
	*s = append(*s, text)
	return s
}

func (s *MailItems) Attachment() *MailItems {
	*s = append(*s, attachment)
	return s
}

func (s *MailItems) All() *MailItems {
	return s.Date().From().To().Cc().Subject().Text().Attachment()
}

func (s *MailItems) has(tgt mailItem) bool {
	for _, i := range *s {
		if i == tgt {
			return true
		}
	}
	return false
}

func (s *MailItems) hasTextORAttachment() bool {
	for _, i := range *s {
		if i == text || i == attachment {
			return true
		}
	}
	return false
}
