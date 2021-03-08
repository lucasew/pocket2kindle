package epub

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
    URL url.URL
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
            articles[i].Content = buf.String()
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()
            articles[i].fetchImages(ctx, book)
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

func GetExtension(name string) string {
    noQueryStrings := strings.Split(name, "?")[0]
    parts := strings.Split(noQueryStrings, ".")
    extension := parts[len(parts) - 1] // the thing after .
    if len(extension) > 5 {
        return ""
    } else {
        return extension
    }
}

// fetchImages return the content with fixed references to fetched images in the book
func (a *EpubArticle) fetchImages(ctx context.Context, book *ep.Epub) {
    images := sync.Map{}
    ch := a.GetImagesFromHtml()
    heartbeat := time.NewTicker(time.Second)
    stuck := 0
    defer heartbeat.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case img := <-ch:
            stuck = 0
            if img == "" {
                continue
            }
            imgUrl, err := url.Parse(img)
            if err != nil {
                log.Printf("Cant parse url '%s'", imgUrl.String())
                continue
            }
            imgUrl = a.URL.ResolveReference(imgUrl)
            _, isAlreadyHere := images.Load(imgUrl)
            if isAlreadyHere {
                continue
            }
            r, err := http.Get(imgUrl.String())
            if (r != nil && r.Body != nil) {
                r.Body.Close()
            }
            if err != nil {
                log.Printf("Not importing image '%s' cause %s", img, err)
                continue
            }
            if r == nil || r.StatusCode < 200 || r.StatusCode > 400 {
                log.Printf("Not importing image '%s' cause status code is %d", img, r.StatusCode)
                continue
            }
            extension := GetExtension(imgUrl.String())
            if extension == "" {
                log.Printf("'%s' ignored: no extension", imgUrl)
                continue
            }
            newName := fmt.Sprintf("%s.%s", uuid.New().String(), extension)
            switch strings.ToLower(extension) {
            case "jpg", "jpeg", "png", "svg":
                newName, err = book.AddImage(imgUrl.String(), newName)
                if err != nil {
                    log.Printf("Failed to add image '%s': %s", imgUrl.String(), err)
                    continue
                }
            default:
                log.Printf("Extension '%s' for the image '%s' is not known for images", extension, imgUrl.String())
                continue
            }
            log.Printf("Importing image '%s' as '%s'", imgUrl, newName)
            images.Store(imgUrl, newName)
            a.Content = strings.ReplaceAll(a.Content, img, newName)
        case <-heartbeat.C:
            if len(ch) == 0 && stuck >= 5 {
                return
            }
            stuck++
        }
    }
}


func (a *EpubArticle) GetImagesFromHtml() chan(string) {
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
                        if len(strings.Trim(v.Val, " ")) > 0 {
                            ch <- v.Val
                        }
                        break
                    }
                }
            }
            recur(node.FirstChild)
            recur(node.NextSibling)
        }
        buf := bytes.NewBufferString(a.Content)
        node, err := html.Parse(buf)
        if err != nil {
            return
        }
        recur(node)
    }()
    return ch
}
