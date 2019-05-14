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
}

// Criteria ...
// Go 1.9 (August 2017) の新機能 type alias を使用
// 'type A B'と 'type A = B' は異なるので注意
// ユーザーが imap パッケージを import 不要となる
// .
type Criteria = imap.SearchCriteria

// CreateClient ...
// Don't forget 'defer Client.Logout()'
// .
func CreateClient(ci *ConnInfo) (*Client, error) {
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

// fetchMessage ...
// ch, done := make(chan *imap.Message), make(chan error)
// go func() { done <- c.fetchMessage(criteria, ch) }()
// if err := <- done; err != nil{
// 		return err
// }
// .
func (c *Client) fetchMessage(name string, criteria *Criteria, ch chan *imap.Message) error {
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
	log.Printf("Found [%v] message(s).\n", len(seqNums))

	items := []imap.FetchItem{c.section.FetchItem()}
	seqset := new(imap.SeqSet)
	seqset.AddNum(seqNums...)
	// ch は c.imapClient.Fetch 内部で close される
	return c.imapClient.Fetch(seqset, items, ch)
}

func (c *Client) messageToMail(dst chan *Mail, src chan *imap.Message, items *MailItems) error {
	// src は送信元で close される
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

		if items.has(subject) {
			nm.Subject, err = h.Subject()
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

// Fetch ...
// .
func (c *Client) Fetch(name string, criteria *Criteria, items *MailItems, ch chan *Mail) error {
	ch0, done0 := make(chan *imap.Message), make(chan error, 1)
	// ch0 は fetchMessage 内で close される
	go func() { done0 <- c.fetchMessage(name, criteria, ch0) }()

	done := make(chan error, 1)
	// ch は messageToMail 内で close される
	go func() { done <- c.messageToMail(ch, ch0, items) }()

	if err := <-done0; err != nil {
		return err
	}
	if err := <-done; err != nil {
		return err
	}
	return nil
}

func mailToAttachment(dst chan *Attachment, src chan *Mail) {
	defer close(dst)
	// src は送信元で close
	for m := range src {
		for _, a := range m.Attachments {
			dst <- a
		}
	}
}

// FetchAttachment ...
// .
func (c *Client) FetchAttachment(name string, criteria *Criteria, ch chan *Attachment) error {
	// 原因調査中だが buffer = 0 だと deadlock する
	ch0 := make(chan *Mail, 1)
	// ch0 は Fetch 内で close
	if err := c.Fetch(name, criteria, NewMailItems().Attachment(), ch0); err != nil {
		return err
	}
	// ch は mailToAttachment 内で close
	mailToAttachment(ch, ch0)
	return nil
}
