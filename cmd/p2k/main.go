package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"time"

	// "github.com/davecgh/go-spew/spew"
	readability "github.com/go-shiori/go-readability"
	"github.com/lucasew/pocket2kindle/pocket"
	"github.com/motemen/go-pocket/api"
)

var client pocket.PocketActor

const favoriteUrl = "https://getpocket.com/v3/send?actions=%5B%7B%22action%22%3A%22favorite%22%2C%22time%22%3A[TIME]%2C%22item_id%22%3A[ITEM_ID]%7D%5D&access_token=[ACCESS_TOKEN]&consumer_key=[CONSUMER_KEY]"

// flags
var articleCount int
var timeout int
var dontDeleteIntermediates bool
var kindleEmail string
var archiveBundled bool

func init() {
    log.Printf("Initializing...")
    flag.IntVar(&articleCount, "c", 10, "how much articles to fetch")
    flag.IntVar(&timeout, "t", 30, "timeout to fetch all articles")
    flag.BoolVar(&dontDeleteIntermediates, "d", false, "dont delete intermediate files")
    flag.StringVar(&kindleEmail, "k", "", "kindle email to send things")
    flag.BoolVar(&archiveBundled, "a", false, "archive bundled articles in pocket")
    flag.Parse()
    client = pocket.NewPocketActor(MustGetenv("POCKET_CONSUMER_KEY"), MustGetenv("POCKET_REQUEST_TOKEN"))
    MustLookupBinary("ebook-convert") // calibre
}

type Article struct {
    Pocket api.Item
    Readability readability.Article
    FavUrl string
}

func main() {
    baseCtx := context.Background()
    timeout := time.Duration(timeout)*time.Second
    timeoutCtx, cancel := context.WithTimeout(baseCtx, timeout)
    defer cancel()
    articles, err := ParseArticleStream(timeoutCtx)
    if err != nil {
        log.Fatal(err)
    }
    var articleIds []int
    articleIds, err = ArticleToEpub(timeoutCtx, articles, "__book__.epub")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Converting ebook to mobi using calibre...")
    cmd := exec.Command("ebook-convert", "__book__.epub", "out.mobi")
    defer DeleteIntermediate("__book__.epub")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Env = os.Environ()
    err = cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
    if kindleEmail != "" {
        err := SendEmail(kindleEmail)
        if err != nil {
            log.Fatal(err)
        }
    }
    if archiveBundled {
        log.Printf("Archiving bundled articles in pocket...")
        archiveAction := make([]*api.Action, len(articleIds))
        for i, article := range articleIds {
            archiveAction[i] = &api.Action{
                Action: "archive",
                ItemID: article,
            }
        }
        _, err = client.GetSuper().Modify(archiveAction...)
        if err != nil {
            log.Fatal(err)
        }
    }
    log.Printf("Done!")
}
