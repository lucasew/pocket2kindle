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
	// "github.com/davecgh/go-spew/spew"
	readability "github.com/go-shiori/go-readability"
	"github.com/google/uuid"
	"github.com/lucasew/p2k/pocket"
	"github.com/motemen/go-pocket/api"
	html "golang.org/x/net/html"
)

var client pocket.PocketActor

const favoriteUrl = "https://getpocket.com/v3/send?actions=%5B%7B%22action%22%3A%22archive%22%2C%22time%22%3A[TIME]%2C%22item_id%22%3A[ITEM_ID]%7D%5D&access_token=[ACCESS_TOKEN]&consumer_key=[CONSUMER_KEY]"

// flags
var articleCount int
var timeout int
var dontDeleteIntermediates bool

func init() {
    log.Printf("Initializing...")
    flag.IntVar(&articleCount, "c", 10, "how much articles to fetch")
    flag.IntVar(&timeout, "t", 30, "timeout to fetch all articles")
    flag.BoolVar(&dontDeleteIntermediates, "d", false, "dont delete intermediate files")
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

func DeleteIntermediate(file string) {
    if !dontDeleteIntermediates {
        log.Printf("Deleting intermediate file '%s'...", file)
        os.RemoveAll(file)
    }
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
    var err error
    for item := range articles {
        select {
        case <-ctx.Done():
            break
        default:
            var content string
            log.Printf("Adding article '%s' to the book", item.Pocket.ResolvedTitle)
            FillFavURL(&item)
            content, err = GetContent(item)
            if err != nil {
                log.Printf("Book chapter generation failed for '%s': %s", item.Pocket.ResolvedTitle, err.Error())
                continue
            }
            content, err = FetchImages(content, book)
            if err != nil {
                log.Printf("Error when fetching image: %s", err.Error())
            }
            book.AddSection(content, item.Pocket.ResolvedTitle, uuid.New().String(), "")
            i++
        }
    }
    log.Printf("Creating a epub with %d articles", i)
    return book.Write(epubFile)
}

func FillFavURL(article *Article) {
    article.FavUrl = favoriteUrl
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[ACCESS_TOKEN]", client.GetSuper().AccessToken)
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[CONSUMER_KEY]", client.GetSuper().ConsumerKey)
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[TIME]", fmt.Sprintf("%d", time.Now().Unix()))
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[ITEM_ID]", fmt.Sprintf("%d", article.Pocket.ItemID))
}

func GetExtension(name string) string {
    parts := strings.Split(name, ".")
    return parts[len(parts) - 1] // the thing after .
}

// &{Parent:0xc000688700 FirstChild:<nil> LastChild:<nil> PrevSibling:<nil> NextSibling:<nil> Type:3 DataAtom:img Data:img Namespace: Attr:[{Namespace: Key:src Val:https://uploads.sitepoint.com/wp-content/uploads/2020/12/1606848785desktop_perf.svg} {Namespace: Key:alt Val:Chart showing simulation time for first 10 days on desktop} {Namespace: Key:loading Val:lazy}]}
func FetchImages(content string, book *epub.Epub) (processed string, err error) {
    var downloaders sync.WaitGroup
    images := make(chan([2]string))
    var handleDownload = func (node *html.Node) {
        defer downloaders.Done()
        if node.Data == "img" {
            for _, v := range node.Attr {
                if v.Key == "src" {
                    log.Printf("Downloading image '%s'", v.Val)
                    filename := fmt.Sprintf("%s.%s", uuid.New(), GetExtension(v.Val))
                    filename, err = book.AddImage(v.Val, filename)
                    if err != nil {
                        break
                    }
                    images <- [2]string{v.Val, filename}
                    break
                }
            }
        } else {
            return
        }
    }
    var recur func(node *html.Node) error
    recur = func (node *html.Node) (err error) {
        if node == nil {
            return nil
        }
        if node.Data == "img" {
            downloaders.Add(1)
            go handleDownload(node)
        }
        err = recur(node.FirstChild)
        if err != nil {
            return err
        }
        return recur(node.NextSibling)
    }
    buf := bytes.NewBufferString(content)
    nodes, err := html.Parse(buf)
    if err != nil {
        return
    }
    err = recur(nodes)
    if err != nil {
        return
    }
    go func() {
        downloaders.Wait()
        close(images)
    }()
    for item := range images {
        content = strings.ReplaceAll(content, item[0], item[1])
    }
    downloaders.Wait()
    return content, nil
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
    ret := buf.String()
    // println(ret)
    return ret, nil
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
    }()
    go func() {
        parserWg.Wait()
        close(parsedItems)
    }()
    for k, item := range items.List {
        parserWg.Add(1)
        go func(k string, item api.Item) {
            if atomic.LoadInt32(&i) > int32(articleCount) {
                close(parsedItems)
                return
            }
            log.Printf("Processing '%s'...", item.ResolvedTitle)
            defer parserWg.Done()
            article, err := readability.FromURL(item.ResolvedURL, time.Second*10)
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
            log.Printf("Processed '%s'!", item.ResolvedTitle)

        }(k, item)
    }
    return parsedItems, nil
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
    err = ArticleToEpub(timeoutCtx, articles, "__book__.epub")
    if err != nil {
        log.Fatal(err)
    }
    cmd := exec.Command("ebook-convert", "__book__.epub", "out.mobi")
    defer DeleteIntermediate("__book__.epub")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Env = os.Environ()
    err = cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
}
