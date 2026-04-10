// main.go
package main

import (
	"os"

	"github.com/mattiasthalen/qlik-sync/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
