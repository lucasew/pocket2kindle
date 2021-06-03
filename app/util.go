package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
    "errors"
)

var ErrNonZeroExitCode = errors.New("non zero exit code")

func (a *App) LookupBinary(name string) error {
    a.Printf("Looking for '%s'", name)
    _, err := exec.LookPath(name)
    if err != nil {
        return fmt.Errorf("%e: %s", ErrExecutableNotFound, name)
    }
    return nil
}


func (a *App) DeleteIntermediate(file string) {
    if !a.DontDeleteIntermediates {
        log.Printf("Deleting intermediate file '%s'...", file)
        os.RemoveAll(file)
    }
}

func RunCommandContext(ctx context.Context, cmd *exec.Cmd) error {
    err := cmd.Start()
    if err != nil {
        return err
    }
    cb := make(chan(error))
    go func() {
        closed := false
        go func() {
            <-ctx.Done()
            if closed {
                return
            }
            closed = true
            cmd.Process.Kill()
            cb <- context.Canceled
            close(cb)
        }()
        go func() {
            p, err := cmd.Process.Wait()
            closed = true
            if p.ExitCode() != 0 {
                cb <- fmt.Errorf("%w: %d", ErrNonZeroExitCode, p.ExitCode())
            } else {
                cb <- err
            }
            close(cb)
        }()
    }()
    return <-cb
}
