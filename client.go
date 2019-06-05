package mailg

import (
	"log"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"

	// ISO-2022-JP で encode された日本語メールを読むのに必要
	_ "github.com/emersion/go-message/charset"
)

const (
	errLimit int = 3
)

type Client struct {
	imapClient *client.Client
	connInfo   *ConnInfo
	section    *imap.BodySectionName
}

// Criteria ...
// Go 1.9 (August 2017) の新機能 type alias を使用
// "type A B" と "type A = B" は異なるので注意
// ユーザーが imap パッケージを import 不要となる
// .
type Criteria = imap.SearchCriteria

// Login ...
// Don't forget "defer Client.Logout()"
// .
func Login(ci *ConnInfo) (*Client, error) {
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
// .
func (c *Client) fetchMessage(
	done <-chan interface{},
	name string,
	criteria *Criteria,
) <-chan *imap.Message {
	// 読み取り専用（readOnly: true）で開く
	if _, err := c.imapClient.Select(name, true); err != nil {
		return nil
	}
	log.Printf("Selected [%v].\n", name)

	const timeFormat string = "06/01/02 00:00 MST"
	log.Printf("Search Criteria is [%v] ~ [%v].\n",
		criteria.Since.Format(timeFormat), criteria.Before.Format(timeFormat))

	seqNums, err := c.imapClient.Search(criteria)
	if err != nil {
		return nil
	}
	log.Printf("Found [%v] message(s).\n", len(seqNums))

	items := []imap.FetchItem{c.section.FetchItem()}
	seqset := new(imap.SeqSet)
	seqset.AddNum(seqNums...)
	return c.generateMessage(done, seqset, items)
}

// Fetch ...
// done := make(chan interface{})
// defer close(done)
// mailStream := c.Fetch(done, name, criteria, mailg.NewMailItems().All())
// .
func (c *Client) Fetch(
	done <-chan interface{},
	name string,
	criteria *Criteria,
	items *MailItems,
) <-chan *Mail {
	messageStream := c.fetchMessage(done, name, criteria)
	return c.toMail(done, messageStream, items, errLimit)
}

// FetchAttachment ...
// done := make(chan interface{})
// defer close(done)
// attachedStream := c.FetchAttachment(done, name, criteria)
// .
func (c *Client) FetchAttachment(
	done <-chan interface{},
	name string,
	criteria *Criteria,
) <-chan *Attachment {
	messageStream := c.fetchMessage(done, name, criteria)
	return toAttachment(done, c.toMail(done, messageStream, NewMailItems().Attachment(), errLimit))
}
