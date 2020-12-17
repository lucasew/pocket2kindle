package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/bmaupin/go-epub"
	"github.com/google/uuid"
	"golang.org/x/net/html"
)
func ArticleToEpub(ctx context.Context, articles chan(Article), epubFile string) ([]int, error) {
    log.Printf("Starting ebook creation...")
    now := time.Now()
    book := epub.NewEpub(fmt.Sprintf("Pocket articles %d/%d/%d %d:%d", now.Day(), now.Month(), now.Year(), now.Hour(), now.Minute()))
    book.SetAuthor("Some machine, somewhere")
    articleIds := make([]int, 0, articleCount * 3)
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
            articleIds = append(articleIds, item.Pocket.ItemID)
            i++
        }
    }
    log.Printf("Creating a epub with %d articles", i)
    return articleIds, book.Write(epubFile)
}

func FillFavURL(article *Article) {
    article.FavUrl = favoriteUrl
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[ACCESS_TOKEN]", client.GetSuper().AccessToken)
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[CONSUMER_KEY]", client.GetSuper().ConsumerKey)
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[TIME]", fmt.Sprintf("%d", time.Now().Unix()))
    article.FavUrl = strings.ReplaceAll(article.FavUrl, "[ITEM_ID]", fmt.Sprintf("%d", article.Pocket.ItemID))
}

func GetExtension(name string) string {
    noQueryStrings := strings.Split(name, "?")[0]
    parts := strings.Split(noQueryStrings, ".")
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
                    extension := GetExtension(v.Val)
                    if extension == "" {
                        break
                    }
                    filename := fmt.Sprintf("%s.%s", uuid.New(), extension)
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

<a href="{{ .FavUrl }}">Favorite</a>
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
