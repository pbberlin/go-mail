package main

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// ExampleRaw does not use the package;
// and does not use attachments
func ExampleRaw() {

	senderHorst, err := initStuff()
	if err != nil {
		return
	}

	sendIt := func(relayHorst RelayHorst) {

		bf := &bytes.Buffer{}

		// headers start
		// twice - once here - and then again inside the body
		commaSep := strings.Join(relayHorst.Filter(to), ", ") // "To: alice@abc.com, bob@123.com \r\n"
		fmt.Fprintf(bf, "To: %v \r\n", commaSep)
		fmt.Fprint(bf, mimeHTML1)
		fmt.Fprint(bf, mimeHTML2)
		fmt.Fprintf(bf, getSubject(subject, relayHorst.HostNamePort)+"\r\n") // subject is the last header?
		// headers end

		fmt.Fprint(bf, "\r\n")

		fmt.Fprint(bf, getBody(senderHorst, true))

		log.Printf("sending via %s... to %v", relayHorst.HostNamePort, relayHorst.Filter(to))
		err := smtp.SendMail(
			relayHorst.HostNamePort,
			getAuth(relayHorst.HostNamePort), // smtp.Auth interface
			getFrom(senderHorst).Address,     // from
			relayHorst.Filter(to),            // twice - once here - and above in the headers
			bf.Bytes(),
		)
		if err != nil {
			log.Printf(" error sending raw-email via %v:\n\t%v", relayHorst, err)
		} else {
			log.Printf(" raw-email sent")
		}
	}

	for _, relayHorst := range relayHorsts {
		sendIt(relayHorst)
	}

}
