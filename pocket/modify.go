package pocket

import (
    api "github.com/motemen/go-pocket/api"
)

type Action = api.Action

var NewArchiveAction = api.NewArchiveAction

func (p PocketActor) Modify(actions ...*Action) error {
    _, err := p.super.Modify(actions...)
    return err
}
