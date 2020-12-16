package main

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/motemen/go-pocket/api"
)

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
