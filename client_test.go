package mailg_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/nekonbu72/mailg"
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

	Date     string `json:"date"`
	From     string `json:"from"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Text     string `json:"text"`
	FileName string `json:"fileName"`
	FileText string `json:"fileText"`

	Criteria *mailg.Criteria
}

const (
	jsonpath string = "test.json"
)

func createMyTest() *MyTest {
	mt := new(MyTest)
	if err := sjson.OpenDecode(jsonpath, mt); err != nil {
		panic("")
	}
	since, _ := time.Parse(mt.TimeFormat, mt.SinceDay)
	before := since.AddDate(0, 0, mt.DaysDuration)
	mt.Criteria = &mailg.Criteria{Since: since, Before: before}
	return mt
}

func createClient(mt *MyTest) *mailg.Client {
	ci := &mailg.ConnInfo{
		Host:     mt.Host,
		Port:     mt.Port,
		User:     mt.User,
		Password: mt.Password,
	}

	c, err := mailg.CreateClient(ci)
	if err != nil {
		panic("")
	}
	return c
}

func createMyTestClient() (*MyTest, *mailg.Client) {
	mt := createMyTest()
	return mt, createClient(mt)
}

func TestCreateClient(t *testing.T) {
	mt := createMyTest()
	ci := &mailg.ConnInfo{
		Host:     mt.Host,
		Port:     mt.Port,
		User:     mt.User,
		Password: mt.Password,
	}
	c, err := mailg.CreateClient(ci)

	defer func() {
		if err := c.Logout(); err != nil {
			t.Errorf("Logout: %v\n", err)
		}
	}()

	if err != nil {
		t.Errorf("CreateClient: %v\n", err)
	}
}

func TestFetch(t *testing.T) {
	mt, c := createMyTestClient()
	defer c.Logout()

	done := make(chan interface{})
	defer close(done)
	ch := c.Fetch(done, mt.Name, mt.Criteria, mailg.NewMailItems().All())

	var ms []*mailg.Mail
	for m := range ch {
		ms = append(ms, m)
	}

	if len(ms) != 1 {
		t.Errorf("Fetch: %v\n", len(ms))
		return
	}

	if ms[0].Date.Format(mt.TimeFormat) != mt.Date {
		t.Errorf("Date: %v\n", ms[0].Date.Format(mt.TimeFormat))
		return
	}

	if ms[0].From[0] != mt.From {
		t.Errorf("From: %v\n", ms[0].From[0])
		return
	}

	if ms[0].To[0] != mt.To {
		t.Errorf("To: %v\n", ms[0].To[0])
		return
	}

	if ms[0].Subject != mt.Subject {
		t.Errorf("Subject: %v\n", ms[0].Subject)
		return
	}

	if ms[0].Text != mt.Text {
		t.Errorf("Text: %v\n", ms[0].Text)
		return
	}
}

func TestFetchAttachment(t *testing.T) {
	mt, c := createMyTestClient()
	defer c.Logout()

	done := make(chan interface{})
	defer close(done)
	ch := c.FetchAttachment(done, mt.Name, mt.Criteria)

	var as []*mailg.Attachment
	for a := range ch {
		as = append(as, a)
	}

	if len(as) != 1 {
		t.Errorf("FetchAttachment: %v\n", len(as))
		return
	}

	a := as[0]
	if a.Filename != mt.FileName {
		t.Errorf("FileName: %v\n", a.Filename)
		return
	}

	bs, err := ioutil.ReadAll(a.Reader)
	if err != nil {
		t.Errorf("ReadAll: %v\n", err)
		return
	}
	if string(bs) != mt.FileText {
		t.Errorf("FileText: %v\n", string(bs))
		return
	}
}
