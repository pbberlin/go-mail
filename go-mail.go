// Package email allows to send emails with attachments.
// en.wikipedia.org/wiki/MIME
// inspired by github.com/scorredoira/email
package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var baseChrOnly = regexp.MustCompile(`[^a-zA-Z0-9_]`) // everything non ascii
var multiHyphen = regexp.MustCompile("[-]{2,}")       // more than one hyphen

// first three unused
const (
	mimeHdr1  = "MIME-version: 1.0;\r\n"
	mimeHdr2a = "Content-Type: text/plain; charset=\"UTF-8\";\r\n"
	mimeHdr2b = "Content-Type: text/html; charset=\"UTF-8\";\r\n"
	frontier  = "f46d043c813270fc6b04c2d223da"
)

// Header - custom email header.
type Header struct {
	Key   string
	Value string
}

// Attachment - a file attachment.
type Attachment struct {
	Filename string
	Data     []byte
	Inline   bool
}

// Message - an smtp message.
type Message struct {
	From          mail.Address
	To            []string
	Cc            []string
	Bcc           []string
	ReplyTo       string
	CustomHeaders []Header

	Subject string

	ContentType string // of body
	Encoding    string // for main body part
	Body        string // should this be a byte slice?

	Attachments []*Attachment
}

func newMessage(subject, body, contentType, encoding string) *Message {
	m := &Message{
		Subject:     subject,
		Body:        body,
		ContentType: contentType,
		Encoding:    encoding,
	}
	m.Attachments = []*Attachment{}
	return m
}

// NewMessagePlain - message with possible attachments
func NewMessagePlain(subject string, body string) *Message {
	return newMessage(
		subject,
		body,
		"text/plain",
		"UTF-8",
	)
}

// NewMessageHTML - message with possible attachments
func NewMessageHTML(subject string, body string) *Message {
	return newMessage(
		subject,
		body,
		"text/html",
		"UTF-8",
	)
}

//
//

func (m *Message) AddTo(address mail.Address) []string {
	m.To = append(m.To, address.String())
	return m.To
}

func (m *Message) AddCc(address mail.Address) []string {
	m.Cc = append(m.Cc, address.String())
	return m.Cc
}

func (m *Message) AddBcc(address mail.Address) []string {
	m.Bcc = append(m.Bcc, address.String())
	return m.Bcc
}

// Tolist returns all the recipients of the email
func (m *Message) Tolist() []string {
	rcptList := []string{}

	toList, _ := mail.ParseAddressList(strings.Join(m.To, ","))
	for _, to := range toList {
		rcptList = append(rcptList, to.Address)
	}

	ccList, _ := mail.ParseAddressList(strings.Join(m.Cc, ","))
	for _, cc := range ccList {
		rcptList = append(rcptList, cc.Address)
	}

	bccList, _ := mail.ParseAddressList(strings.Join(m.Bcc, ","))
	for _, bcc := range bccList {
		rcptList = append(rcptList, bcc.Address)
	}

	return rcptList
}

// AddCustomHeader add header to message
func (m *Message) AddCustomHeader(key string, value string) Header {
	newHeader := Header{Key: key, Value: value}
	m.CustomHeaders = append(m.CustomHeaders, newHeader)
	return newHeader
}

//
// attachment stuff

// cleanseFN removes all non ASCII chars from a filename
func cleanseFN(fn string) string {
	// chop off file extension; for later re-appending
	xt := filepath.Ext(fn)
	fn = fn[:len(fn)-len(xt)]
	// for instance fn = "Äößss__ss8a&sadfq"
	fn = baseChrOnly.ReplaceAllString(fn, "-")
	fn = multiHyphen.ReplaceAllString(fn, "-")
	fn = strings.Trim(fn, "-")
	return fn + xt
}

func (m *Message) attach(file string, inline bool) error {

	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	filename := filepath.Base(file)
	filename = cleanseFN(filename)

	att := Attachment{
		Filename: filename,
		Data:     data,
		Inline:   inline,
	}

	m.Attachments = append(m.Attachments, &att)

	return nil
}

// Attach attaches a file.
func (m *Message) Attach(file string) error {
	return m.attach(file, false)
}

// AttachInline -  includes file as inline attachment;
// not implemented
func (m *Message) AttachInline(file string) error {
	return m.attach(file, true)
}

// AttachByteSlice - for binary attachment.
func (m *Message) AttachByteSlice(filename string, buf []byte, inline bool) error {
	att := Attachment{
		Filename: cleanseFN(filename),
		Data:     buf,
		Inline:   inline,
	}
	m.Attachments = append(m.Attachments, &att)
	return nil
}

//
// render to email

// Bytes returns the mail data
func (m *Message) Bytes() []byte {

	bf := bytes.NewBuffer(nil)

	bf.WriteString("From: " + m.From.String() + "\r\n")

	bf.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")

	bf.WriteString("To: " + strings.Join(m.To, ",") + "\r\n")

	if len(m.Cc) > 0 {
		bf.WriteString("Cc: " + strings.Join(m.Cc, ",") + "\r\n")
	}

	if len(m.ReplyTo) > 0 {
		bf.WriteString("Reply-To: " + m.ReplyTo + "\r\n")
	}

	for _, header := range m.CustomHeaders {
		fmt.Fprintf(bf, "%s: %s\r\n", header.Key, header.Value)
	}

	// subject header - with encoding
	var coder = base64.StdEncoding
	var subject = "=?UTF-8?B?" + coder.EncodeToString([]byte(m.Subject)) + "?="
	bf.WriteString("Subject: " + subject + "\r\n")

	// simple structure without attachments
	if len(m.Attachments) == 0 {
		fmt.Fprintf(bf, "MIME-Version: 1.0\r\n")
		fmt.Fprintf(bf, "Content-Type: %s; charset=\"%v\";\r\n", m.ContentType, m.Encoding)
		fmt.Fprint(bf, "\r\n") // end of headers
		fmt.Fprint(bf, m.Body)
		return bf.Bytes()
	}

	// distinct structure for attachments
	fmt.Fprintf(bf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(bf, "Content-Type: multipart/mixed; boundary=%v\r\n", frontier)
	fmt.Fprint(bf, "\r\n") // end of headers
	fmt.Fprint(bf, "Fallback message for old clients: Multiple parts email in MIME format.")
	fmt.Fprint(bf, "\r\n") // end of body
	fmt.Fprintf(bf, "--%v\r\n", frontier)
	// block 1 - email text body
	fmt.Fprintf(bf, "Content-Type: %s; charset=\"%v\";\r\n", m.ContentType, m.Encoding)
	fmt.Fprint(bf, "\r\n") // end sub-headers 1
	fmt.Fprint(bf, m.Body)
	fmt.Fprint(bf, "\r\n") // end of body

	for _, attachment := range m.Attachments {
		fmt.Fprintf(bf, "--%v\r\n", frontier)

		// if attachment.Inline {
		// 	fmt.Fprintf(bf, "Content-Type: message/rfc822\r\n")
		// 	fmt.Fprintf(bf, "Content-Disposition: inline; filename=%v\r\n\r\n", clearString(attachment.Filename))
		// 	fmt.Fprint(bf, attachment.Data)
		// }

		ext := filepath.Ext(attachment.Filename)
		mimetype := mime.TypeByExtension(ext)
		if mimetype != "" {
			mime := fmt.Sprintf("Content-Type: %s\r\n", mimetype)
			fmt.Fprint(bf, mime)
		} else {
			fmt.Fprint(bf, "Content-Type: application/octet-stream\r\n")
		}
		fmt.Fprint(bf, "Content-Transfer-Encoding: base64\r\n")

		//  the filename encoding was replaced by filename *cleansing*
		//   filenames of attachments should not contain fancy characters
		// 	buf.WriteString("Content-Disposition: attachment; filename=\"=?UTF-8?B?")
		// 	buf.WriteString(coder.EncodeToString([]byte(attachment.Filename)))
		// 	buf.WriteString("?=\"\r\n\r\n")

		fmt.Fprintf(bf, "Content-Disposition: attachment; filename=%v;\r\n", cleanseFN(attachment.Filename))
		fmt.Fprint(bf, "\r\n") // end sub-headers 1

		b := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Data)))
		base64.StdEncoding.Encode(b, attachment.Data)
		// write base64 content in lines of up to 76 chars;
		// disgustingly ineffective
		for i, l := 0, len(b); i < l; i++ {
			bf.WriteByte(b[i])
			if (i+1)%76 == 0 {
				bf.WriteString("\r\n")
			}
		}
		fmt.Fprint(bf, "\r\n") // end of body

	}

	fmt.Fprintf(bf, "--%v--\r\n", frontier) // end of *all* parts

	return bf.Bytes()
}

// Send sends the message.
func Send(addr string, auth smtp.Auth, m *Message) error {
	return smtp.SendMail(addr, auth, m.From.Address, m.Tolist(), m.Bytes())
}
