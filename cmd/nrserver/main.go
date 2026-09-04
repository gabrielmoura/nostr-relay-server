/*
Copyright © 2024 Gabriel Moura <gmouradev96@gmail.com>
*/
package main

import (
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/cmd"
	"github.com/gabrielmoura/nostr-relay-server/internal/version"
)

func main() {
	fmt.Println(version.Get().String())
	cmd.Execute()
}
