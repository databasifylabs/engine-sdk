package main

import (
	"os"

	sdk "github.com/databasifylabs/engine-sdk"
	"github.com/databasifylabs/engine-sdk/host"
)

func main() {
	os.Exit(host.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, sdk.Version))
}
