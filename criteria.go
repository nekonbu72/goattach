package goattach

import (
	"errors"
	"time"

	"github.com/emersion/go-imap"
)

type Criteria struct {
	Name   string
	Since  time.Time
	Before time.Time
}

func (c *Criteria) CreateImapSearch() (*imap.SearchCriteria, error) {
	if !c.isValid() {
		return nil, errors.New("goattach: criteria fields error")
	}
	return &imap.SearchCriteria{Since: c.Since, Before: c.Before}, nil
}

func (c *Criteria) isValid() bool {
	if c.Name == "" || c.Since.IsZero() || c.Before.IsZero() {
		return false
	}
	return true
}
