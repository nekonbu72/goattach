package goattach

import "io"

type Attachment struct {
	Filename string
	Reader   io.Reader
}
