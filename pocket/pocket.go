package pocket

import (
    api "github.com/motemen/go-pocket/api"
)

type PocketActor struct {
    super *api.Client
}

func NewPocketActor(consumerKey, accessToken string) PocketActor {
    return PocketActor{
        super: api.NewClient(consumerKey, accessToken),
    }
}

func (super PocketActor) GetSuper() *api.Client {
    return super.super
}

