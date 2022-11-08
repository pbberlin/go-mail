package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	gm "github.com/zew/go-mail"
)

type Recipient struct {
	Email    string `csv:"email"`
	Sex      int    `csv:"sex"`
	Title    string `csv:"title"`
	Lastname string `csv:"lastname"`
	NoMail   string `csv:"!Mail !Call"`

	Link     template.HTML `csv:"link"` // avoid escaping
	Language string        `csv:"lang"`

	Anrede          string `csv:"anrede"`
	MonthYear       string `csv:"-"` // Oktober 2022, Octover 2022
	FullClosingDate string `csv:"-"` // Friday, 17th October 2022   Freitag, den 17. Oktober 2022,
}

func (r *Recipient) SetDerived() {
	if r.Language == "de" {
		if r.Sex == 1 {
			r.Anrede = "Sehr geehrter Herr "
		}
		if r.Sex == 2 {
			r.Anrede = "Sehr geehrte Frau "
		}
		if r.Title != "" {
			r.Anrede += r.Title + " "
		}
		r.Anrede += r.Lastname
	}
	if r.Language == "en" {
		if r.Sex == 1 {
			r.Anrede = "Dear Mr. "
		}
		if r.Sex == 2 {
			r.Anrede = "Dear Ms. "
		}
		if r.Title != "" {
			r.Anrede = "Dear " + r.Title + " "
		}
		r.Anrede += r.Lastname
	}

	// now := time.Now()
	// now = now.AddDate(0, 0, 5)
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		loc = time.FixedZone("UTC_-2", -2*60*60)
	}
	now := time.Date(2022, 11, 11, 17, 0, 0, 0, loc)
	// now = time.Date(2022, 11, 11+3, 17, 0, 0, 0, loc)

	y := now.Year()
	m := now.Month()
	w := now.Weekday()
	r.MonthYear = fmt.Sprintf("%v %v", MonthByInt(int(m), r.Language), y)

	r.FullClosingDate = now.Format("Monday, 2. January 2006")

	if r.Language == "de" {
		r.FullClosingDate =
			strings.Replace(
				r.FullClosingDate,
				MonthByInt(int(m), "en"),
				MonthByInt(int(m), "de"),
				-1,
			)

		r.FullClosingDate =
			strings.Replace(
				r.FullClosingDate,
				WeekdayByInt(int(w), "en"),
				WeekdayByInt(int(w), "de"),
				-1,
			)

		r.FullClosingDate += "," // add apposition comma
	}

}

var relayZimbra = RelayHorst{
	HostNamePort: "zimbra.zew.de:25",
	Internal:     false,
	External:     true,
}

func getText(rec Recipient, survey, process, language string) (subject, body string) {
	fn := fmt.Sprintf("%v-%v.md", process, language)
	pth := filepath.Join(".", "tpl", survey, fn)
	t, err := template.ParseFiles(pth)
	if err != nil {
		log.Fatalf("could not parse templates; %v", err)
	}

	// log.Printf("template parse success %v", t.Name())

	sb := &strings.Builder{}
	t.ExecuteTemplate(sb, fn, rec)

	if strings.Contains(sb.String(), "\r\n") {
		log.Fatalf("template %v contains \"r\"n - should be only \"n", t.Name())
	}

	lines := strings.Split(sb.String(), "\n")

	return lines[0], strings.Join(lines[1:], "\n")
}

func singleEmail(rec Recipient) {

	m := gm.NewMessagePlain(getText(rec, "fmt", "invitation", rec.Language))
	// 	m = gm.NewMessageHTML(getSubject(subject, relayHorst.HostNamePort), getBody(senderHorst, true))
	log.Printf("  subject: %v", m.Subject)
	// log.Print(m.Body)
	// return

	m.From = mail.Address{}
	m.From.Name = "Finanzmarkttest"
	m.From.Address = "noreply@zew.de"
	m.To = []string{rec.Email}

	m.ReplyTo = "finanzmarkttest@zew.de"
	// return-path is a hidden email header
	// indicating where bounced emails will be processed.
	m.AddCustomHeader("Return-Path", m.ReplyTo)

	// todo undeliverable emails
	// m.

	//
	// attachments
	if false {
		if err := m.Attach("./attachments/1.pdf"); err != nil {
			log.Fatal(err)
		}
	}

	m.AddCustomHeader("X-Mailer", "go-mail")

	log.Printf("  sending via %s... to %v", relayZimbra.HostNamePort, rec.Lastname)
	err := gm.Send(
		relayZimbra.HostNamePort,
		getAuth(relayZimbra.HostNamePort),
		m,
	)
	if err != nil {
		log.Printf(" error sending lib-email  %v:\n\t%v", relayZimbra, err)
	} else {
		// log.Printf("  lib-email sent")
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
		rec.SetDerived()
		if idx1 > 5 || idx1 < len(recipients)-5 {
			continue
		}
		log.Printf(
			"record %v - %12v  %26v  %v",
			idx1,
			rec.MonthYear,
			rec.FullClosingDate,
			rec.Anrede,
		)
	}

	// back to start of file
	if _, err := inFile.Seek(0, 0); err != nil {
		log.Print(err)
		return err
	}

	for idx1, rec := range recipients {
		log.Printf("#%03v - %2v - %1v - %10v %-16v - %-32v ",
			idx1+1,
			rec.Language, rec.Sex,
			rec.Title, rec.Lastname,
			rec.Email,
		)
		if strings.Contains(rec.NoMail, "noMail") {
			log.Printf("  skipping 'noMail'")
			continue
		}
		// singleEmail(*rec)
		time.Sleep(time.Second / 5)
	}

	return nil
}
