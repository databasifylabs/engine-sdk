package sdk

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/ctl"
	"github.com/databasifylabs/engine-sdk/host"
	"github.com/databasifylabs/engine-sdk/layout"
	"github.com/databasifylabs/engine-sdk/supervise"
)

const Version = "1.0.0"

const usageText = `usage: databasify <command>

commands:
  serve                       supervise the processes in $DATABASIFY_HOME/conf/processes.json
  ctl <verb> [options] [unit] systemctl, systemd-run and journalctl style control
`

func Run() int {
	args := os.Args[1:]
	if len(args) == 0 {
		return usage(os.Stderr)
	}
	if onHost() {
		return host.Main(args, os.Stdin, os.Stdout, os.Stderr)
	}
	paths := layout.FromEnv()
	switch args[0] {
	case "serve":
		return serve(paths)
	case "ctl":
		return ctl.CLI{Socket: paths.Socket(), LogFile: paths.LogFile, Stdout: os.Stdout, Stderr: os.Stderr}.Run(args[1:])
	case "operation-run":
		return runOperation(args[1:])
	}
	if command.IsWellFormed(args) {
		return command.Run(context.Background(), Version, args, os.Stdin, os.Stdout)
	}

	return usage(os.Stderr)
}

func onHost() bool {
	return os.Getenv("DATABASIFY_ENGINE") == ""
}

func usage(out io.Writer) int {
	_, _ = io.WriteString(out, usageText)

	return ctl.ExitUsage
}

func serve(paths layout.Layout) int {
	if err := supervise.BecomeSubreaper(); err != nil {
		fmt.Fprintf(os.Stderr, "databasify: cannot become subreaper: %v\n", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	supervisor := supervise.New(supervise.Options{Layout: paths, Console: os.Stdout})
	listener, err := ctl.Listen(paths.Socket())
	if err != nil {
		fmt.Fprintf(os.Stderr, "databasify: cannot listen on %s: %v\n", paths.Socket(), err)

		return ctl.ExitInternal
	}
	supervisor.Boot()
	go reloadOnHangup(ctx, supervisor)
	go func() {
		if err := ctl.NewServer(supervisor, paths.LogFile).Serve(ctx, listener); err != nil {
			fmt.Fprintf(os.Stderr, "databasify: control socket stopped: %v\n", err)
		}
	}()
	<-ctx.Done()
	supervisor.Shutdown()

	return 0
}

func reloadOnHangup(ctx context.Context, supervisor *supervise.Supervisor) {
	hangups := make(chan os.Signal, 1)
	signal.Notify(hangups, syscall.SIGHUP)
	defer signal.Stop(hangups)
	for {
		select {
		case <-ctx.Done():
			return
		case <-hangups:
			if err := supervisor.ReloadManifest(); err != nil {
				fmt.Fprintf(os.Stderr, "databasify: reload failed: %v\n", err)
			}
		}
	}
}

func runOperation(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: databasify operation-run <key>")

		return ctl.ExitUsage
	}

	return command.RunOperation(context.Background(), args[0])
}
