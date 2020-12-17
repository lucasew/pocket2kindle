package main

import (
	"log"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/scorredoira/email"
)

func SendEmail(addr ...string) error {
    log.Printf("Setting up email...")
    server := MustGetenv("SMTP_SERVER")
    user := MustGetenv("SMTP_USER")
    passwd := MustGetenv("SMTP_PASSWD")
    m := email.NewMessage("", "")
    m.From = mail.Address{Name: "p2k bot", Address: user}
    m.To = addr
    err := m.Attach("./out.mobi")
    if err != nil {
        return err
    }
    auth := smtp.PlainAuth("", user, passwd, strings.Split(server, ":")[0])
    log.Printf("Sending email...")
    return email.Send(server, auth, m)
}
