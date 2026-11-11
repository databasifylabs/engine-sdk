package host

import (
	"fmt"
	"io"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/ctl"
)

const usageText = `usage: databasify <command>

commands:
  host deploy|remove|tune|restore  manage the engine container on this server
  <group> <verb>                   run an engine command in the engine container
  version                          print the SDK version
`

func Main(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, version string) int {
	if IsHostCommand(args) {
		return RunHostCommand(args, stdin, stdout)
	}
	if command.IsWellFormed(args) {
		return Forward(args, stdin, stdout, stderr)
	}
	if len(args) == 1 && args[0] == "version" {
		_, _ = fmt.Fprintln(stdout, version)

		return 0
	}
	_, _ = io.WriteString(stderr, usageText)

	return ctl.ExitUsage
}
