package epub

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"text/template"

	ep "github.com/bmaupin/go-epub"
	"github.com/google/uuid"
	"golang.org/x/net/html"
)

type EpubArticle struct {
    Title string
    Content string
    Actions map[string]string // Clickable links in the begin and end of each article
}

type EpubOptions struct {
    Title string
    Author string
    Template string
    SendMoreLink string
}

const DefaultArticleTemplate = `
<h1>{{ .Title }}</h1>

{{ range $key, $value := .Actions }}
    <a href="{{ $value }}">{{ $key }}</a><br>
{{ end }}

{{ .Content }}

{{ range $key, $value := .Actions }}
    <a href="{{ $value }}">{{ $key }}</a><br>
{{ end }}
`

func CreateEpub(articles []EpubArticle, options EpubOptions, filename string) error {
    book := ep.NewEpub(options.Title)
    book.SetAuthor(options.Author)
    tplTxt := options.Template
    if tplTxt == "" {
        tplTxt = DefaultArticleTemplate
    }
    tpl, err := template.New("epub").Parse(tplTxt)
    if err != nil {
        return err
    }
    var wg sync.WaitGroup
    wg.Add(len(articles))
    for i := 0; i < len(articles); i++ {
        go func(i int) {
            buf := bytes.NewBuffer([]byte{})
            tpl.Execute(buf, articles[i])
            content := buf.String()
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()
            articles[i].Content = fetchImages(ctx, content, book)
            wg.Done()
        }(i)
    }
    wg.Wait()
    for _, article := range articles {
        filename := fmt.Sprintf("%s.html", uuid.New())
        _, err := book.AddSection(article.Content, article.Title, filename, "")
        if err != nil {
            return err
        }
    }
    if options.SendMoreLink != "" {
        data := fmt.Sprintf(`<h1>Final actions</h1> <a href="%s">Send More</a>`, options.SendMoreLink)
        _, err := book.AddSection(data, "Final actions", "finalactions.html", "")
        if err != nil {
            return err
        }
    }
    return book.Write(filename)
}

// fetchImages return the content with fixed references to fetched images in the book
func fetchImages(ctx context.Context, content string, book *ep.Epub) string {
    images := sync.Map{}
    var err error
    ch := GetImagesFromHtml(content)
    heartbeat := time.NewTicker(time.Second)
    defer heartbeat.Stop()
    for {
        select {
        case <-ctx.Done():
            return content
        case img := <-ch:
            _, isAlreadyHere := images.Load(img)
            if isAlreadyHere {
                continue
            }
            newName := fmt.Sprintf("%s.png", uuid.New().String())
            newName, err = book.AddImage(img, newName)
            if err != nil {
                continue
            }
            log.Printf("Importing image '%s' as '%s'", img, newName)
            images.Store(img, newName)
            content = strings.ReplaceAll(content, img, newName)
        case <-heartbeat.C:
            if len(ch) == 0 {
                return content
            }
        }
    }
}

func GetImagesFromHtml(content string) chan(string) {
    ch := make(chan(string), 16)
    go func() {
        defer close(ch)
        var recur func(node *html.Node)
        recur = func(node *html.Node) {
            if node == nil {
                return
            }
            if node.Data == "img" {
                for _, v := range node.Attr {
                    if v.Key == "src" {
                        ch <- v.Val
                        break
                    }
                }
            }
            recur(node.FirstChild)
            recur(node.NextSibling)
        }
        buf := bytes.NewBufferString(content)
        node, err := html.Parse(buf)
        if err != nil {
            return
        }
        recur(node)
    }()
    return ch
}
