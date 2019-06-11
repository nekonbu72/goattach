package mailg

import (
	"errors"
	"time"

	"github.com/emersion/go-imap"

	"github.com/nekonbu72/sjson/sjson"
)

type Setting struct {
	ConnInfo *ConnInfo `json:"connInfo"`
	Criteria *Criteria `json:"criteria"`
}

type ConnInfo struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func NewSetting(path string) (*Setting, error) {
	s := new(Setting)
	err := sjson.OpenDecode(path, s)
	if err != nil {
		return nil, err
	}
	return s, err
}

func (c *ConnInfo) address() (string, error) {
	if !c.isValid() {
		return "", errors.New("mailg: conninfo fields error")
	}

	const delimiter string = ":"
	return c.Host + delimiter + c.Port, nil
}

func (c *ConnInfo) isValid() bool {
	if c.Host == "" || c.Port == "" || c.User == "" || c.Password == "" {
		return false
	}
	return true
}

type Criteria struct {
	Name     string   `json:"name"`
	Duration Duration `json:"duration"`
}

type Duration struct {
	Layout string `json:"layout"`
	Since  string `json:"since"`
	Before string `json:"before"`
}

func (c *Criteria) serachCriteria() (*imap.SearchCriteria, error) {
	since, err := time.Parse(c.Duration.Layout, c.Duration.Since)
	if err != nil {
		return nil, err
	}

	before, err := time.Parse(c.Duration.Layout, c.Duration.Before)
	if err != nil {
		return nil, err
	}

	return &imap.SearchCriteria{Since: since, Before: before}, nil
}
