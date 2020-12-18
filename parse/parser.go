package parser

import (
	// "bytes"
	"context"
	// "io"
	"log"
	// "net/http"
	"time"

	"github.com/go-shiori/go-readability"
)

type ParsedArticle = readability.Article

func Parse(ctx context.Context, url string) (ParsedArticle, error) {
    log.Printf("Parsing '%s'", url)
    // buf := bytes.NewBuffer([]byte{})
    // res, err := http.NewRequestWithContext(ctx, "GET", url, buf)
    // if err != nil {
    //     return ParsedArticle{}, err
    // }
    // resBuf := bytes.NewBuffer([]byte{})
    // _, err = io.Copy(resBuf, res.Body)
    // if err != nil {
    //     return ParsedArticle{}, err
    // }
    // return readability.FromReader(resBuf, url)
    return readability.FromURL(url, 20*time.Second)
}
