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

// Login ...
// Don't forget "defer Client.Logout()"
// .
func Login(connInfo *ConnInfo) (*Client, error) {
	addr, err := connInfo.address()
	if err != nil {
		return nil, err
	}

	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}
	log.Printf("Connected to [%v].\n", addr)

	if err := c.Login(connInfo.User, connInfo.Password); err != nil {
		return nil, err
	}
	log.Printf("Logged in as [%v].\n", connInfo.User)

	return &Client{imapClient: c, connInfo: connInfo, section: new(imap.BodySectionName)}, nil
}

func (c *Client) Logout() error {
	log.Printf("Logged out as [%v].\n", c.connInfo.User)
	return c.imapClient.Logout()
}

// fetchMessage ...
// .
func (c *Client) fetchMessage(
	done <-chan interface{},
	criteria *Criteria,
) <-chan *imap.Message {
	dummyMessageStream := make(chan *imap.Message)
	defer close(dummyMessageStream)

	sc, err := criteria.serachCriteria()
	if err != nil {
		log.Printf("serachCriteria: %v\n", err)
		return dummyMessageStream
	}

	// 読み取り専用（readOnly: true）で開く
	if _, err := c.imapClient.Select(criteria.Name, true); err != nil {
		log.Printf("Select: %v\n", err)
		return dummyMessageStream
	}
	log.Printf("Selected [%v].\n", criteria.Name)

	log.Printf("Search Criteria is [%v] ~ [%v].\n",
		criteria.Duration.Since, criteria.Duration.Before)

	seqNums, err := c.imapClient.Search(sc)
	if err != nil {
		log.Printf("Search: %v\n", err)
		return dummyMessageStream
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
	criteria *Criteria,
	items *MailItems,
) <-chan *Mail {
	messageStream := c.fetchMessage(done, criteria)
	return c.toMail(done, messageStream, items, errLimit)
}

// FetchAttachment ...
// done := make(chan interface{})
// defer close(done)
// attachedStream := c.FetchAttachment(done, name, criteria)
// .
func (c *Client) FetchAttachment(
	done <-chan interface{},
	criteria *Criteria,
) <-chan *Attachment {
	messageStream := c.fetchMessage(done, criteria)
	return toAttachment(done, c.toMail(done, messageStream, NewMailItems().Attachment(), errLimit))
}
