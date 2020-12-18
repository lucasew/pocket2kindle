package epub

import (
    "strings"
)

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

