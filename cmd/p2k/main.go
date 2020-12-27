package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/lucasew/pocket2kindle/app"
)

var a app.App

func init() {
    log.Printf("Initializing...")
    a.Logger = log.New(os.Stderr, log.Prefix(), log.Flags())
    flag.IntVar(&a.ArticleCount, "c", 10, "how much articles to fetch")
    flag.IntVar(&a.Timeout, "t", 30, "timeout to fetch all articles")
    flag.BoolVar(&a.DontDeleteIntermediates, "d", false, "dont delete intermediate files")
    flag.StringVar(&a.KindleEmail, "k", "", "kindle email to send things")
    flag.BoolVar(&a.ArchiveBundled, "a", false, "archive bundled articles in pocket")
    flag.Parse()
    a.PocketConsumerKey = os.Getenv("POCKET_CONSUMER_KEY")
    a.PocketRequestToken = os.Getenv("POCKET_REQUEST_TOKEN")
    if a.KindleEmail != "" {
        a.SMTPServer = MustGetenv("SMTP_SERVER")
        a.SMTPUser = MustGetenv("SMTP_USER")
        a.SMTPPassword = MustGetenv("SMTP_PASSWD")
    }
}

func main() {
    err := a.Run(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Done!")
}

func MustGetenv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        log.Fatalf("environment variable not found: %s", key)
    }
    return v
}

