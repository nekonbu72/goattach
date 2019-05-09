package goattach

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/nekonbu72/sjson/sjson"
)

type MyTest struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`

	TimeFormat   string `json:"timeFormat"`
	SinceDay     string `json:"sinceDay"`
	DaysDuration int    `json:"daysDuration"`

	Name string `json:"name"`

	Filename string `json:"filename"`
	Text     string `json:"text"`
}

const (
	jsonpath string = "test.json"
)

func TestCreateClientLoggedIn(t *testing.T) {
	mt := new(MyTest)
	if err := sjson.OpenDecode(jsonpath, mt); err != nil {
		t.Errorf("OpenDecode: %v\n", err)
	}

	ci := &ConnInfo{
		Host:     mt.Host,
		Port:     mt.Port,
		User:     mt.User,
		Password: mt.Password,
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

func TestFetchMessages(t *testing.T) {
	mt := new(MyTest)
	if err := sjson.OpenDecode(jsonpath, mt); err != nil {
		t.Errorf("OpenDecode: %v\n", err)
	}

	c, err := CreateClientLoggedIn(&ConnInfo{
		Host:     mt.Host,
		Port:     mt.Port,
		User:     mt.User,
		Password: mt.Password,
	})
	if err != nil {
		t.Errorf("CreateClientLoggedIn: %v\n", err)
	}
	defer c.Logout()

	since, _ := time.Parse(mt.TimeFormat, mt.SinceDay)
	before := since.AddDate(0, 0, mt.DaysDuration)
	criteria := &Criteria{
		Name:   mt.Name,
		Since:  since,
		Before: before,
	}

	ch, done := newChanFetchMessages()
	go func() { done <- c.fetchMessages(criteria, ch) }()
	if err := <-done; err != nil {
		t.Errorf("fetch: %v\n", err)
	}
}

func TestFetchAttachments(t *testing.T) {
	mt := new(MyTest)
	if err := sjson.OpenDecode(jsonpath, mt); err != nil {
		t.Errorf("OpenDecode: %v\n", err)
	}

	c, err := CreateClientLoggedIn(&ConnInfo{
		Host:     mt.Host,
		Port:     mt.Port,
		User:     mt.User,
		Password: mt.Password,
	})
	if err != nil {
		t.Errorf("CreateClientLoggedIn: %v\n", err)
	}
	defer c.Logout()

	since, _ := time.Parse(mt.TimeFormat, mt.SinceDay)
	before := since.AddDate(0, 0, mt.DaysDuration)
	criteria := &Criteria{
		Name:   mt.Name,
		Since:  since,
		Before: before,
	}

	ch, done := NewChanFetchAttachments(1)
	go func() { done <- c.FetchAttachments(criteria, ch) }()
	var as []*Attachment
	for a := range ch {
		as = append(as, a)
	}

	if err := <-done; err != nil {
		t.Errorf("FetchAttachments: %v\n", err)
	}

	if len(as) != 1 {
		t.Error("FetchAttachments")
	}

	for _, a := range as {
		if a.Filename != mt.Filename {
			t.Errorf("Filename: %v\n", "")
		}

		bs, err := ioutil.ReadAll(a.Reader)
		if err != nil {
			t.Errorf("ReadAll: %v\n", err)
		}
		if string(bs) != mt.Text {
			t.Errorf("text: %v\n", "")
		}
	}
}
