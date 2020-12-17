package main

import (
	"log"
	"net/smtp"
	"os"
	"strings"

	"github.com/domodwyer/mailyak"
)

func SendEmail(addr ...string) error {
    log.Printf("Setting up email...")
    mobi, err := os.Open("./out.mobi")
    if err != nil {
        return err
    }
    defer mobi.Close()
    server := MustGetenv("SMTP_SERVER")
    user := MustGetenv("SMTP_USER")
    auth := smtp.PlainAuth("", user, MustGetenv("SMTP_PASSWD"), strings.Split(server, ":")[0])
    mail := mailyak.New(server, auth)
    mail.To(addr...)
    mail.From(user)
    mail.Subject("")
    mail.AttachInlineWithMimeType("ebook.mobi", mobi, "application/x-mobipocket-ebook")
    log.Printf("Sending email...")
    err = mail.Send()
    if err != nil {
        return err
    }
    return nil
}
