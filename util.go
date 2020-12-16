package main

import (
	"log"
	"os"
	"os/exec"
)

func MustLookupBinary(command string) {
    _, err := exec.LookPath(command)
    if err != nil {
        log.Fatalf("command not fount: %s", command)
    }
}

func MustGetenv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        log.Fatalf("environment variable not found: %s", key)
    }
    return v
}

func DeleteIntermediate(file string) {
    if !dontDeleteIntermediates {
        log.Printf("Deleting intermediate file '%s'...", file)
        os.RemoveAll(file)
    }
}
