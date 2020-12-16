package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	epub "github.com/bmaupin/go-epub"
	readability "github.com/go-shiori/go-readability"
	"github.com/google/uuid"
	"github.com/lucasew/p2k/pocket"
	"github.com/motemen/go-pocket/api"
)

var client pocket.PocketActor

const favoriteUrl = "https://getpocket.com/v3/send?actions=%5B%7B%22action%22%3A%22archive%22%2C%22time%22%3A[TIME]%2C%22item_id%22%3A[ITEM_ID]%7D%5D&access_token=[ACCESS_TOKEN]&consumer_key=[CONSUMER_KEY]"

// flags
var articleCount int
var timeout int

func init() {
    log.Printf("Initializing...")
    flag.IntVar(&articleCount, "c", 10, "how much articles to fetch")
    flag.IntVar(&timeout, "t", 30, "timeout to fetch all articles")
    flag.Parse()
    client = pocket.NewPocketActor(MustGetenv("POCKET_CONSUMER_KEY"), MustGetenv("POCKET_REQUEST_TOKEN"))
    MustLookupBinary("ebook-convert") // calibre
}

func MustLookupBinary(command string) {
    _, err := exec.LookPath(command)
    if err != nil {
        log.Fatalf("command not fount: %s", command)
    }
}

func MustGetenv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        log.Fatalf("environment variable not found: %s", key)
    }
    return v
}

type Article struct {
    Pocket api.Item
    Readability readability.Article
    FavUrl string
}

func ArticleToEpub(ctx context.Context, articles chan(Article), epubFile string) error {
    log.Printf("Starting ebook creation...")
    now := time.Now()
    book := epub.NewEpub(fmt.Sprintf("Pocket articles: %d %d %d", now.Day(), now.Month(), now.Year()))
    book.SetAuthor("Some machine, somewhere")
    i := 0
    for item := range articles {
        select {
        case <-ctx.Done():
            break
        default:
            log.Printf("Adding article '%s' to the book", item.Pocket.ResolvedTitle)
            content, err := GetContent(item)
            if err != nil {
                log.Printf("Book chapter generation failed for '%s': %s", item.Pocket.ResolvedTitle, err.Error())
                continue
            }
            book.AddSection(content, item.Pocket.ResolvedTitle, uuid.New().String(), "")
            i++
        }
    }
    log.Printf("Creating a epub with %d articles", i)
    return book.Write("book.epub")
}

func FillFavURL(article Article) Article {
    article.FavUrl = favoriteUrl
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[ACCESS_TOKEN]", client.GetSuper().AccessToken)
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[CONSUMER_KEY]", client.GetSuper().ConsumerKey)
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[TIME]", fmt.Sprintf("%d", time.Now().Unix()))
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[ITEM_ID]", fmt.Sprintf("%d", article.Pocket.ItemID))
    return article
}

var contentTemplate = template.Must(template.New("content").Parse(`
<h1>{{ .Readability.Title }}</h1>
<a href="{{ .FavUrl }}">Favorite</a>

{{ .Readability.Content }}
`))

func GetContent(article Article) (string, error) {
    buf := bytes.NewBuffer([]byte{})
    err := contentTemplate.Execute(buf, article)
    if err != nil {
        return "", err
    }
    return buf.String(), nil
}

func ParseArticleStream(ctx context.Context) (chan(Article), error) {
    options := api.RetrieveOption{
        State: api.StateUnread,
        Count: articleCount * 3,
        DetailType: api.DetailTypeComplete,
        Favorite: api.FavoriteFilterUnfavorited,
    }
    log.Printf("Getting data from pocket...")
    items, err := client.GetSuper().Retrieve(&options)
    if err != nil {
        return nil, err
    }
    parsedItems := make(chan(Article), articleCount)
    end := false
    var parserWg sync.WaitGroup
    var i int32 = 0
    go func() {
        <-ctx.Done()
        end = true
        close(parsedItems)
    }()
    for k, item := range items.List {
        go func(k string, item api.Item) {
            if atomic.LoadInt32(&i) > int32(articleCount) {
                close(parsedItems)
                return
            }
            log.Printf("Processing %s...", k)
            parserWg.Add(1)
            defer parserWg.Done()
            article, err := readability.FromURL(item.ResolvedURL, time.Second*30)
            if err != nil {
                log.Printf("Error processing %s: %s", k, err.Error())
                return
            }
            if end {
                return
            }
            parsedItems <- Article{
                Pocket: item,
                Readability: article,
            }
            atomic.AddInt32(&i, 1)
            log.Printf("Processed %s!", k)

        }(k, item)
    }
    return parsedItems, nil
}

func main() {
    baseCtx := context.Background()
    timeout := time.Duration(timeout)*time.Second
    timeoutCtx, cancel := context.WithTimeout(baseCtx, timeout)
    articles, err := ParseArticleStream(timeoutCtx)
    if err != nil {
        log.Fatal(err)
    }
    err = ArticleToEpub(timeoutCtx, articles, "book.epub")
    if err != nil {
        log.Fatal(err)
    }
    defer cancel()
}
