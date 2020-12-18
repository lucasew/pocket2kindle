package pocket

import (
	"context"
	"log"
	"time"

	"github.com/motemen/go-pocket/api"
)

type PocketRetriever struct {
    stop bool
    client PocketActor
    ch chan(api.Item)
    offset int
    options PocketRetrieveOption
}

type PocketRetrieveOption = api.RetrieveOption

func (p *PocketRetriever) getRound() *api.RetrieveResult {
    for {
        options := p.options
        options.Offset = p.offset
        items, err := p.client.super.Retrieve(&options)
        if err != nil {
            log.Printf("Failed to fetch data from pocket. Trying in 5 seconds: %s", err.Error())
            time.Sleep(5*time.Second)
        }
        p.offset += options.Count
        return items
    }
}

func (p *PocketRetriever) run() {
    defer close(p.ch)
    for {
        round := p.getRound()
        for _, item := range round.List {
            if p.stop {
                return
            }
            p.ch <- item
        }
    }
}

func (p *PocketRetriever) Next(ctx context.Context) *api.Item {
    select {
    case <-ctx.Done():
        return nil
    case a := <-p.ch:
        return &a
    }
}

func (p *PocketRetriever) Close() {
    p.stop = true
}

func (super PocketActor) Retrieve(options api.RetrieveOption) *PocketRetriever {
    itemChan := make(chan(api.Item), 3)
    retriever := &PocketRetriever{
        stop: false,
        ch: itemChan,
        client: super,
        offset: 0,
        options: options,
    }
    go retriever.run()
    return retriever
}
