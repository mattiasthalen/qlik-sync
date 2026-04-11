// main.go
package main

import (
	"errors"
	"os"

	"github.com/mattiasthalen/qlik-sync/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, cmd.ErrPartialSync) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
