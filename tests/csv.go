package main

import (
	"encoding/csv"
	"io"
	"log"
	"net/mail"
	"os"

	"github.com/gocarina/gocsv"
	gm "github.com/zew/go-mail"
)

type Recipient struct {
	Email    string `csv:"email"`
	Sex      string `csv:"sex"`
	Title    string `csv:"title"`
	Lastname string `csv:"lastname"`

	Link     string `csv:"link"`
	Language string `csv:"lang"`

	Salutation string `csv:"anrede"`
}

/*
	    	Concat(CASE sex
	                WHEN 1 THEN 'Sehr geehrter Herr '
	                WHEN 2 THEN 'Sehr geehrte Frau '
	              END, title,
	              CASE
                    WHEN Length(title) > 0 THEN ' '
                    ELSE ''
                  END, lastname)                              AS 'anrede',


                CASE title
                WHEN ''    THEN
                    CASE sex
                        WHEN 1     THEN 'Dear Mr. '
                        WHEN 2     THEN 'Dear Ms. '
                    END
                ELSE
                    concat('Dear ',title)
                END,
                lastname

*/

var relayZimbra = RelayHorst{
	HostNamePort: "zimbra.zew.de:25",
	Internal:     false,
	External:     true,
}

func getText(survey, process, language string) (subject, body string) {
	return "subject", "body"
}

func sendSingle(rec Recipient) {

	m := gm.NewMessagePlain(getText("fmt", "invitation", rec.Language))
	// 	m = gm.NewMessageHTML(getSubject(subject, relayHorst.HostNamePort), getBody(senderHorst, true))

	m.From = mail.Address{}
	m.From.Name = "Finanzmarkttest"
	m.From.Address = "noreply@zew.de"
	m.To = []string{rec.Email}
	m.ReplyTo = "finanzmarkttest@zew.de"

	// todo undeliverable emails
	// m.

	//
	// attachments
	if false {
		if err := m.Attach("./attachments/1.pdf"); err != nil {
			log.Fatal(err)
		}
	}

	m.AddCustomHeader("X-CUSTOMER-id", "xxxxx")

	log.Printf("sending via %s... to %+v", relayZimbra.HostNamePort, rec)
	err := gm.Send(
		relayZimbra.HostNamePort,
		getAuth(relayZimbra.HostNamePort),
		m,
	)
	if err != nil {
		log.Printf(" error sending lib-email  %v:\n\t%v", relayZimbra, err)
	} else {
		log.Printf(" lib-email sent")
	}
}

func ProcessCSV() error {

	inFile, err := os.OpenFile(
		"./csv/fmt-invitation.csv",
		os.O_RDWR|os.O_CREATE,
		os.ModePerm,
	)
	if err != nil {
		log.Print(err)
		return err
	}
	defer inFile.Close()

	recipients := []*Recipient{}

	// set option for gocsv lib
	// use semicolon as delimiter
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = ';'
		// r.LazyQuotes = true
		// r.TrimLeadingSpace = true
		return r
	})

	if err := gocsv.UnmarshalFile(inFile, &recipients); err != nil {
		log.Print(err)
		return err
	}

	for idx1, rec := range recipients {
		if idx1 > 5 || idx1 < len(recipients)-5 {
			continue
		}
		log.Printf("record %v - %+v", idx1, rec)
	}

	// back to start of file
	if _, err := inFile.Seek(0, 0); err != nil {
		log.Print(err)
		return err
	}

	for idx1, rec := range recipients {
		log.Printf("sending #%03v - %2v - %1v - %10v %-16v - %-32v ",
			idx1+1,
			rec.Language, rec.Sex,
			rec.Title, rec.Lastname,
			rec.Email,
		)
		// sendSingle(*rec)
	}

	return nil
}
