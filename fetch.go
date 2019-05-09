package goattach

import (
	"bytes"
	"errors"
	"io"
	"log"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type Client struct {
	imapClient *client.Client
	connInfo   *ConnInfo
	section    *imap.BodySectionName
}

// CreateClientLoggedIn ...
// Don't forget 'defer Client.Logout()'
// .
func CreateClientLoggedIn(ci *ConnInfo) (*Client, error) {
	addr, err := ci.Address()
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

	return &Client{imapClient: c, connInfo: ci}, nil
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
func (c *Client) fetchMessages(criteria *Criteria, ch chan *imap.Message) error {
	icr, err := criteria.CreateImapSearch()
	if err != nil {
		return err
	}

	if _, err := c.imapClient.Select(criteria.Name, true); err != nil {
		return err
	}
	log.Printf("Selected [%v].\n", criteria.Name)

	const timeFormat string = "06/01/02 00:00 MST"
	log.Printf("Search Criteria is [%v] ~ [%v].\n",
		criteria.Since.Format(timeFormat), criteria.Before.Format(timeFormat))

	seqNums, err := c.imapClient.Search(icr)
	if err != nil {
		return err
	}
	log.Printf("Found [%v] message(s).\n", len(seqNums))

	c.section = new(imap.BodySectionName)
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
func (c *Client) FetchAttachments(criteria *Criteria, ch chan *Attachment) error {
	defer close(ch)
	ch0, done := newChanFetchMessages()
	go func() { done <- c.fetchMessages(criteria, ch0) }()
	for message := range ch0 {
		if err := c.readMessageAsAttachment(message, ch); err != nil {
			continue
		}
	}

	if err := <-done; err != nil {
		return err
	}
	return nil
}

func (c *Client) readMessageAsAttachment(m *imap.Message, ch chan *Attachment) error {
	mailReader, err := mail.CreateReader(m.GetBody(c.section))
	if err != nil {
		return err
	}

	for {
		part, err := mailReader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		a, err := readAttachment(part)
		if err != nil {
			continue
		}
		ch <- a
	}
	return nil
}

func readAttachment(p *mail.Part) (*Attachment, error) {
	attachmentHeader, ok := p.Header.(*mail.AttachmentHeader)
	if !ok {
		// 添付ファイル以外の場合
		return nil, errors.New("Type Assertion not ok")
	}

	fileName, err := attachmentHeader.Filename()
	if err != nil {
		return nil, err
	}

	r, w := io.Pipe()
	done := make(chan error, 1)
	go func() {
		defer w.Close()
		_, err := io.Copy(w, p.Body)
		done <- err
	}()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	if err := <-done; err != nil {
		return nil, err
	}
	return &Attachment{Filename: fileName, Reader: buf}, nil
}
