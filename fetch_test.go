package goattach

import (
	"testing"
	"time"
)

const (
	host     string = "imapgms.jnet.sei.co.jp"
	port     string = "993"
	user     string = "s150209"
	password string = "tomo0101@"

	timeFormat   string = "2006-01-02 MST"
	sinceDay     string = "2019-05-05 JST"
	daysDuration int    = 1

	name string = "998_test"

	outDir = "test\\out"
)

func TestCreateClientLoggedIn(t *testing.T) {
	ci := &ConnInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}

	c, err := CreateClientLoggedIn(ci)

	defer func() {
		if err := c.Logout(); err != nil {
			t.Errorf("Logout: %v\n", err)
		}
	}()

	if err != nil {
		t.Errorf("CreateClientLoggedIn: %v\n", err)
	}
}
func TestFetchAttachmentReaders(t *testing.T) {
	c, err := CreateClientLoggedIn(&ConnInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	})
	if err != nil {
		t.Errorf("CreateClientLoggedIn: %v\n", err)
	}
	defer c.Logout()

	since, _ := time.Parse(timeFormat, sinceDay)
	before := since.AddDate(0, 0, daysDuration)
	criteria := &Criteria{
		Name:   name,
		Since:  since,
		Before: before,
	}

	_, err = c.FetchAttachmentReaders(criteria)
	if err != nil {
		t.Errorf("FetchAttachmentReaders: %v\n", err)
	}
}
