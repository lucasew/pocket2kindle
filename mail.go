package main

import (
	"log"
	"net/smtp"
	"strings"
	"github.com/scorredoira/email"
)

func SendEmail(addr ...string) error {
    log.Printf("Setting up email...")
    m := email.NewMessage("", "")
    m.To = addr
    err := m.Attach("./out.mobi")
    if err != nil {
        return err
    }
    server := MustGetenv("SMTP_SERVER")
    user := MustGetenv("SMTP_USER")
    auth := smtp.PlainAuth("", user, MustGetenv("SMTP_PASSWD"), strings.Split(server, ":")[0])
    log.Printf("Sending email...")
    return email.Send(server, auth, m)
}
