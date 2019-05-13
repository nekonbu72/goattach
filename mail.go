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
	Sub         string
	Text        string
	Attachments []*Attachment
}

type Attachment struct {
	Filename string
	io.Reader
}
