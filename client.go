package mailg

import (
	"bytes"
	"io"
	"io/ioutil"
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

// fetchMessages ...
// ch := make(chan *imap.Message)
// done := make(chan error)
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

func (c *Client) messageToMail(dst chan *Mail, src chan *imap.Message, items *MailItems) error {
	// src は送信元で close してくれる
	defer close(dst)
	for m := range src {
		nm := new(Mail)

		r, err := mail.CreateReader(m.GetBody(c.section))
		if err != nil {
			return err
		}

		h := r.Header

		if items.has(date) {
			nm.Date, err = h.Date()
			if err != nil {
				return err
			}
		}

		if items.has(from) {
			nm.From, err = addressToStr(h.AddressList("From"))
			if err != nil {
				return err
			}
		}

		if items.has(to) {
			nm.To, err = addressToStr(h.AddressList("To"))
			if err != nil {
				return err
			}
		}

		if items.has(cc) {
			nm.Cc, err = addressToStr(h.AddressList("Cc"))
			if err != nil {
				return err
			}
		}

		if items.has(sub) {
			nm.Sub, err = h.Subject()
			if err != nil {
				return err
			}
		}

		if items.hasTextORAttachment() {
			for {
				p, err := r.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				switch h := p.Header.(type) {
				case *mail.InlineHeader:
					if items.has(text) {
						b, err := ioutil.ReadAll(p.Body)
						if err != nil {
							return err
						}
						nm.Text = string(b)
					}
				case *mail.AttachmentHeader:
					if items.has(attachment) {
						fileName, err := h.Filename()
						if err != nil {
							return err
						}

						buf := bytes.NewBuffer(nil)
						_, err = buf.ReadFrom(p.Body)
						if err != nil {
							return err
						}
						nm.Attachments = append(nm.Attachments, &Attachment{Filename: fileName, Reader: buf})
					}
				}
			}
		}
		dst <- nm
	}
	return nil
}

func addressToStr(as []*mail.Address, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}

	var s []string
	for _, a := range as {
		s = append(s, a.Address)
	}
	return s, nil
}

func (c *Client) Fetch(name string, criteria *Criteria, items *MailItems, ch chan *Mail) error {
	ch0 := make(chan *imap.Message)
	done := make(chan error, 1)
	// ch0 は fetchMessages 内で close される
	go func() { done <- c.fetchMessages(name, criteria, ch0) }()

	// ch は messageToMail 内で close される
	if err := c.messageToMail(ch, ch0, items); err != nil {
		return err
	}

	if err := <-done; err != nil {
		return err
	}

	return nil
}

func mailToAttachment(dst chan *Attachment, src chan *Mail) {
	defer close(dst)
	for m := range src {
		for _, a := range m.Attachments {
			dst <- a
		}
	}
}

// FetchAttachments ...
// ch := make(chan *Attachment)
// done := make(chan error)
// go func() { done <- c.FetchAttachments(criteria, ch) }()
// .
func (c *Client) FetchAttachments(name string, criteria *Criteria, ch chan *Attachment) error {
	ch0 := make(chan *Mail)
	done := make(chan error, 1)
	// ch0 は Fetch 内で close
	go func() { done <- c.Fetch(name, criteria, NewMailItems().Attachment(), ch0) }()

	mailToAttachment(ch, ch0)
	if err := <-done; err != nil {
		return err
	}
	return nil
}
