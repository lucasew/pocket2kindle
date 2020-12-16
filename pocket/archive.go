package pocket

import (
    api "github.com/motemen/go-pocket/api"
)

func (p PocketActor) Archive(id int) error {
    action := api.NewArchiveAction(id)
    _, err := p.super.Modify(action)
    return err
}
