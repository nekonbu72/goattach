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

	Filename string `json:"filename"`
	Text     string `json:"text"`

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

	c, err := mailg.CreateClientLoggedIn(ci)
	if err != nil {
		panic("")
	}
	return c
}

func createMyTestClient() (*MyTest, *mailg.Client) {
	mt := createMyTest()
	return mt, createClient(mt)
}

func TestCreateClientLoggedIn(t *testing.T) {
	mt := createMyTest()
	ci := &mailg.ConnInfo{
		Host:     mt.Host,
		Port:     mt.Port,
		User:     mt.User,
		Password: mt.Password,
	}

	c, err := mailg.CreateClientLoggedIn(ci)

	defer func() {
		if err := c.Logout(); err != nil {
			t.Errorf("Logout: %v\n", err)
		}
	}()

	if err != nil {
		t.Errorf("CreateClientLoggedIn: %v\n", err)
	}
}

func TestFetch(t *testing.T) {
	mt, c := createMyTestClient()
	defer c.Logout()

	ch := make(chan *mailg.Mail, 1)
	done := make(chan error)

	go func() { done <- c.Fetch(mt.Name, mt.Criteria, mailg.NewMailItems().All(), ch) }()
	if err := <-done; err != nil {
		t.Errorf("Fetch: %v\n", err)
	}

	var ms []*mailg.Mail
	for m := range ch {
		ms = append(ms, m)
	}

	if len(ms) != 1 {
		t.Error("Fetch")
	}

	a := ms[0].Attachments[0]
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
