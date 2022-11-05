package main

import (
	"bytes"
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

/*
	config data,
	for some email sending task
*/

type RelayHorst struct {
	HostNamePort string
	Internal     bool
	External     bool
}

func (rh RelayHorst) Filter(addresses []string) []string {
	filtered := []string{}
	for _, addr := range addresses {
		internal := strings.Contains(addr, "zew.de")
		if internal && !rh.Internal {
			continue
		}
		if !internal && !rh.External {
			continue
		}
		filtered = append(filtered, addr)
	}
	return filtered
}

var relayHorsts = []RelayHorst{}
var to = []string{}

const (
	mimeHTML1 = "MIME-version: 1.0\r\n"
	mimeHTML2 = "Content-Type: text/html; charset=\"UTF-8\";\r\n"

	subject = "Subject: email test headline"
)

func getSubject(subject, relayHorst string) string {
	return fmt.Sprintf("%v via %v", subject, relayHorst)
}

func initStuff() (string, error) {

	senderHorst, err := os.Hostname()
	if err != nil {
		return "sender-host-no-worki", err
	}
	log.Printf("Sender horst is %v", senderHorst)

	if strings.HasPrefix(senderHorst, "NB-") {
		// from the notebook, behind firewall, ZEW internally
		relayHorsts = []RelayHorst{
			{
				HostNamePort: "email.zew.de:25", // from intern
				Internal:     true,
				External:     false, // all emails must belong to domain zew.de - otherwise 'relay rejection'
			},
			{
				HostNamePort: "hermes.zew-private.de:25", // from intern
				Internal:     true,
				External:     true,
			},
			{
				HostNamePort: "zimbra.zew.de:25",
				Internal:     false,
				External:     true,
			},
		}
		to = []string{
			"peter.buchmann@zew.de",
			"peter.buchmann.68@gmail.com",
			"peter.buchmann@web.de",
		}
	} else {
		// from DMZ, externally
		relayHorsts = []RelayHorst{
			{
				HostNamePort: "hermes.zew.de:25",
				Internal:     true,
				External:     true,
			},
			{
				HostNamePort: "zimbra.zew.de:25",
				Internal:     false,
				External:     true,
			},
		}
		to = []string{
			"peter.buchmann.68@gmail.com",
			"peter.buchmann@web.de",
		}

	}

	return senderHorst, nil
}

func getBody(senderHorst string, html bool) string {
	body := &bytes.Buffer{}

	fmt.Fprint(body, "some<br>body\n<br>to<br>test<br><br>\n\n")
	fmt.Fprintf(body, "<p style='color:#48d1cc;'   >HTML email test sent at %v</p>", time.Now().Format(time.RFC850))
	fmt.Fprintf(body, "<p style='font-weight:bold;'>from %v</p>", senderHorst)

	return body.String()
}

func getAuth(relayHorst string) (auth smtp.Auth) {
	if relayHorst == "zimbra.zew.de:25" {
		// authentication information.
		username := "fmt-relay"
		password := "Qual32!Crux" // yes, you found a password, but its worthless without access to the company network
		auth = smtp.PlainAuth("",
			username,
			password,
			strings.Split(relayHorst, ":")[0],
		)
	}
	return
}

func getFrom(senderHorst string) (addr mail.Address) {
	addr = mail.Address{
		// Name: "Peter Buchmann",
		Name: "Finanzmarkttest",
		// Address: "peter.buchmann@zew.de",
		Address: "finanzmarkttest@zew.de",
	}
	return
}
