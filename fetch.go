package goattach

import (
	"bytes"
	"errors"
	"io"
	"log"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"

	// ISO-2022-JP で encode された日本語メールを読むのに必要
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

type Client struct {
	imapClient *client.Client
	connInfo   *ConnInfo
	section    *imap.BodySectionName
	msgCnt     int
}

// Criteria ...
// Go 1.9 (August 2017) の新機能 type alias を使用
// 'type A B'と 'type A = B' は異なるので注意
// ユーザーが imap パッケージを import 不要となる
// .
type Criteria = imap.SearchCriteria

// CreateClientLoggedIn ...
// Don't forget 'defer Client.Logout()'
// .
func CreateClientLoggedIn(ci *ConnInfo) (*Client, error) {
	addr, err := ci.address()
	if err != nil {
		return nil, err
	}

	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}
	log.Printf("Connected to [%v].\n", addr)

	if err := c.Login(ci.User, ci.Password); err != nil {
		return nil, err
	}
	log.Printf("Logged in as [%v].\n", ci.User)

	return &Client{imapClient: c, connInfo: ci, section: new(imap.BodySectionName)}, nil
}

func (c *Client) Logout() error {
	log.Printf("Logged out as [%v].\n", c.connInfo.User)
	return c.imapClient.Logout()
}

func newChanFetchMessages() (chan *imap.Message, chan error) {
	return make(chan *imap.Message, 10), make(chan error, 1)
}

// fetchMessages ...
// newChanFetchMessages() で生成した chan を
// 引数・返り値にしてゴルーチンで使用する
// go func() { done <- c.fetchMessages(criteria, ch) }()
// .
func (c *Client) fetchMessages(name string, criteria *Criteria, ch chan *imap.Message) error {
	// 読み取り専用（readOnly: true）で開く
	if _, err := c.imapClient.Select(name, true); err != nil {
		return err
	}
	log.Printf("Selected [%v].\n", name)

	const timeFormat string = "06/01/02 00:00 MST"
	log.Printf("Search Criteria is [%v] ~ [%v].\n",
		criteria.Since.Format(timeFormat), criteria.Before.Format(timeFormat))

	seqNums, err := c.imapClient.Search(criteria)
	if err != nil {
		return err
	}
	c.msgCnt = len(seqNums)
	log.Printf("Found [%v] message(s).\n", c.msgCnt)

	items := []imap.FetchItem{c.section.FetchItem()}
	seqset := new(imap.SeqSet)
	seqset.AddNum(seqNums...)

	// c.imapClient.Fetch の三番目の引数 ch は関数内部で close(ch) されている
	return c.imapClient.Fetch(seqset, items, ch)
}

func NewChanFetchAttachments(buffer int) (chan *Attachment, chan error) {
	return make(chan *Attachment, buffer), make(chan error, 1)
}

// FetchAttachments ...
// NewChanFetchAttachments() で生成した chan を
// 引数・返り値にしてゴルーチンで使用する
// go func() { done <- c.FetchAttachments(criteria, ch) }()
// .
func (c *Client) FetchAttachments(name string, criteria *Criteria, ch chan *Attachment) error {
	defer close(ch)
	ch0, done := newChanFetchMessages()
	go func() { done <- c.fetchMessages(name, criteria, ch0) }()
	for m := range ch0 {
		if err := c.fetchAttachment(m, ch); err != nil {
			continue
		}
	}

	if err := <-done; err != nil {
		return err
	}
	return nil
}

func (c *Client) fetchAttachment(m *imap.Message, ch chan *Attachment) error {
	r, err := mail.CreateReader(m.GetBody(c.section))
	if err != nil {
		return err
	}

	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		a, err := toAttachment(p)
		if err != nil {
			continue
		}
		ch <- a
	}
	return nil
}

func toAttachment(p *mail.Part) (*Attachment, error) {
	h, ok := p.Header.(*mail.AttachmentHeader)
	if !ok {
		// 添付ファイル以外の場合
		return nil, errors.New("Type Assertion not ok")
	}

	fileName, err := h.Filename()
	if err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(utilPipe(p.Body, done))
	if err != nil {
		return nil, err
	}

	if err := <-done; err != nil {
		return nil, err
	}
	return &Attachment{Filename: fileName, Reader: buf}, nil
}

func (c *Client) FetchMail(name string, criteria *Criteria, ch chan *Mail) error {

	ch1, done1 := newChanFetchMessages()
	go func() { done1 <- c.fetchMessages(name, criteria, ch1) }()
	// ch2, done2 := newChanPipeMail(c.msgCnt)
	done2 := make(chan error, 1)
	go func() { done2 <- pipeMail(ch, ch1, c.section, Date, From, To, Cc, Sub, Text, Attachments) }()

	if err := <-done1; err != nil {
		return err
	}
	if err := <-done2; err != nil {
		return err
	}
	return nil
}
