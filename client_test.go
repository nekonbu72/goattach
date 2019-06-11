package mailg

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/nekonbu72/sjson/sjson"
)

type TestExpect struct {
	Date     string `json:"date"`
	From     string `json:"from"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Text     string `json:"text"`
	FileName string `json:"fileName"`
	FileText string `json:"fileText"`
}

const (
	testDir = "test"
	test    = "test.json"
	expect  = "expect.json"
)

func newTestSetting() *Setting {
	s, err := NewSetting(path.Join(testDir, test))
	if err != nil {
		panic(err)
	}
	return s
}

func newTextExpect() *TestExpect {
	e := new(TestExpect)
	if err := sjson.OpenDecode(path.Join(testDir, expect), e); err != nil {
		panic(err)
	}
	return e
}

func newTestClient(ci *ConnInfo) *Client {
	c, err := Login(ci)
	if err != nil {
		panic(err)
	}
	return c
}

func TestLogin(t *testing.T) {
	s := newTestSetting()
	c, err := Login(s.ConnInfo)

	defer func() {
		if err := c.Logout(); err != nil {
			t.Errorf("Logout: %v\n", err)
		}
	}()

	if err != nil {
		t.Errorf("Login: %v\n", err)
	}
}

func TestFetch(t *testing.T) {
	s := newTestSetting()
	c := newTestClient(s.ConnInfo)
	defer c.Logout()
	e := newTextExpect()

	done := make(chan interface{})
	defer close(done)
	ch := c.Fetch(done, s.Criteria, NewMailItems().All())

	var ms []*Mail
	for m := range ch {
		ms = append(ms, m)
	}

	if len(ms) != 1 {
		t.Errorf("len: %v\n", len(ms))
		return
	}

	if ms[0].Date.Format(s.Criteria.Duration.Layout) != e.Date {
		t.Errorf("Date: %v\n", ms[0].Date.Format(s.Criteria.Duration.Layout))
		return
	}

	if ms[0].From[0] != e.From {
		t.Errorf("From: %v\n", ms[0].From[0])
		return
	}

	if ms[0].To[0] != e.To {
		t.Errorf("To: %v\n", ms[0].To[0])
		return
	}

	if ms[0].Subject != e.Subject {
		t.Errorf("Subject: %v\n", ms[0].Subject)
		return
	}

	if ms[0].Text != e.Text {
		t.Errorf("Text: %v\n", ms[0].Text)
		return
	}
}

func TestFetchAttachment(t *testing.T) {
	s := newTestSetting()
	c := newTestClient(s.ConnInfo)
	defer c.Logout()
	e := newTextExpect()

	done := make(chan interface{})
	defer close(done)
	ch := c.FetchAttachment(done, s.Criteria)

	var as []*Attachment
	for a := range ch {
		as = append(as, a)
	}

	if len(as) != 1 {
		t.Errorf("len: %v\n", len(as))
		return
	}

	a := as[0]
	if a.FileName != e.FileName {
		t.Errorf("FileName: %v\n", a.FileName)
		return
	}

	bs, err := ioutil.ReadAll(a.Reader)
	if err != nil {
		t.Errorf("ReadAll: %v\n", err)
		return
	}
	if string(bs) != e.FileText {
		t.Errorf("FileText: %v\n", string(bs))
		return
	}
}
