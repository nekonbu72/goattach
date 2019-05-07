package goattach

import (
	"io"
	"io/ioutil"
	"log"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type Client struct {
	imapClient *client.Client
	connInfo   *ConnInfo
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

func (c *Client) FetchAttachmentReaders(criteria *Criteria) ([]*Attachment, error) {
	icr, err := criteria.CreateImapSearch()
	if err != nil {
		return nil, err
	}

	if _, err := c.imapClient.Select(criteria.Name, true); err != nil {
		return nil, err
	}
	log.Printf("Selected [%v].\n", criteria.Name)

	const timeFormat string = "06/01/02 00:00 MST"
	log.Printf("Search Criteria is [%v] ~ [%v].\n",
		criteria.Since.Format(timeFormat), criteria.Before.Format(timeFormat))

	seqNums, err := c.imapClient.Search(icr)
	if err != nil {
		return nil, err
	}
	log.Printf("Found [%v] message(s).\n", len(seqNums))

	section := new(imap.BodySectionName)
	items := []imap.FetchItem{section.FetchItem()}
	seqset := new(imap.SeqSet)
	seqset.AddNum(seqNums...)

	ch := make(chan *imap.Message)
	done := make(chan error, 1)
	// client.Fetch の三番目の引数 ch は関数内部で close(ch) されている
	go func() { done <- c.imapClient.Fetch(seqset, items, ch) }()

	var attachments []*Attachment

	cnt := 0                           // Logging
	cnt2 := 0                          // Logging
	const ans string = "Hello, world!" // test

	for message := range ch {
		cnt++ // Logging
		log.Printf("Fetched message [%v/%v].\n", cnt, len(seqNums))

		mailReader, err := mail.CreateReader(message.GetBody(section))
		if err != nil {
			log.Println("CreateReader:", err)
			continue
		}

		for {
			part, err := mailReader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println("NextPart:", err)
				continue
			}

			attachmentHeader, ok := part.Header.(*mail.AttachmentHeader)
			if !ok {
				continue
			}

			cnt2++ // Logging

			fileName, err := attachmentHeader.Filename()
			if err != nil {
				log.Println("Filename:", err)
				continue
			}

			log.Printf("!!!%p\n", part)      // test
			log.Printf("!!!%p\n", part.Body) // test
			// DebugReadAll("part.Body", &part.Body, ans) // test

			attachment := &Attachment{
				Filename: fileName,
				Reader:   part.Body}
			DebugReadAll("field of struct", &attachment.Reader, ans) // test

			attachments = append(attachments, attachment)
			DebugReadAll("slice inside of loop", &attachments[len(attachments)-1].Reader, ans) // test

			// log.Printf("Fetched attached file [No.%v] from message [%v/%v].\n",
			// 	cnt2, cnt, len(seqNums))
		}
	}

	if err := <-done; err != nil {
		return nil, err
	}

	DebugReadAll("slice outside of loop", &attachments[len(attachments)-1].Reader, ans) // test
	log.Println("")                                                                     // test
	return attachments, nil
}

// DebugReadAll ...
// io.Reader の Debug 用
// .
func DebugReadAll(item string, r *io.Reader, ans string) error {
	bs, err := ioutil.ReadAll(*r)
	if err != nil {
		log.Println("ReadAll:", err)
		return err
	}
	result := string(bs)
	log.Println("")
	log.Printf("[%v]\n", item)
	log.Println("--addr:", r)
	// if result != "" {
	// 	log.Printf("--(contents are next line)\n%v\n", result)
	// } else {
	// 	log.Println("--(no contents)")
	// }
	log.Println("--result:", result == ans)
	return nil
}
