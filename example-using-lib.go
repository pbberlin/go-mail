package gomail

import (
	"fmt"
	"log"
	"strings"
)

func ExampleUsingLib() {

	senderHorst, err := initStuff()
	if err != nil {
		return
	}

	// Send it
	sendIt := func(relayHorst RelayHorst, toRecipient string) {

		// Compose the message
		m := NewMessagePlain(getSubject(subject, relayHorst.HostNamePort), getBody(senderHorst, false))
		if true {
			m = NewMessageHTML(getSubject(subject, relayHorst.HostNamePort), getBody(senderHorst, true))
		}

		m.From = getFrom(senderHorst)
		m.To = []string{toRecipient}

		//
		// attachments
		imgs := []string{"ga1.gif", "ga2.gif", "ga3.gif"}
		for _, fn := range imgs {
			if err := m.Attach(strings.ToUpper(fn), fmt.Sprintf("./attachments/%v", fn), false); err != nil {
				log.Fatal(err)
			}
		}
		if err := m.Attach("PDF1", "./attachments/1.pdf", false); err != nil {
			log.Fatal(err)
		}
		// use Inline to display the attachment inline.
		if err := m.Attach("PDF2", "./attachments/2.pdf", true); err != nil {
			log.Fatal(err)
		}

		m.AddCustomHeader("X-CUSTOMER-id", "xxxxx")

		log.Printf("sending via %s... to %v", relayHorst.HostNamePort, toRecipient)
		err := Send(
			relayHorst.HostNamePort,
			relayHorst.getAuth(),
			m,
		)
		if err != nil {
			log.Printf(" error sending lib-email  %v:\n\t%v", relayHorst, err)
		} else {
			log.Printf(" lib-email sent")
		}
	}

	for _, relayHorst := range relayHorsts {

		toFiltered := relayHorst.Filter(to)
		for _, singleTo := range toFiltered {
			sendIt(relayHorst, singleTo)
		}

	}

}
