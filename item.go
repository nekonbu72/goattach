package gomailpher

type MailItem int

const (
	date MailItem = iota
	from
	to
	cc
	sub
	text
	attachment
)

type MailItems []MailItem

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

func (s *MailItems) Sub() *MailItems {
	*s = append(*s, sub)
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
	return s.Date().From().To().Cc().Sub().Text().Attachment()
}

func (s *MailItems) has(tgt MailItem) bool {
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
