package host

import (
	"io"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/ctl"
)

const usageText = `usage: databasify <command>

commands:
  host deploy|remove  manage the engine container on this server
  <group> <verb>      run a runtime command in the engine container
`

func Main(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if IsHostCommand(args) {
		return RunHostCommand(args, stdin, stdout)
	}
	if command.IsWellFormed(args) {
		return Forward(args, stdin, stdout, stderr)
	}
	_, _ = io.WriteString(stderr, usageText)

	return ctl.ExitUsage
}
