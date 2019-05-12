package goattach

type MailItem int

const (
	Date MailItem = iota
	From
	To
	Cc
	Sub
	Text
	Attachments
)

func hasItem(is []MailItem, tgt MailItem) bool {
	for _, i := range is {
		if i == tgt {
			return true
		}
	}
	return false
}

func hasTextORAttachments(is []MailItem) bool {
	for _, i := range is {
		if i == Text || i == Attachments {
			return true
		}
	}
	return false
}

type MailItems []MailItem
