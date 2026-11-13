package units

import (
	"fmt"
	"strings"

	"github.com/databasifylabs/engine-sdk/ctl"
)

type Control struct {
	Socket string
}

func (c Control) Do(verb string, units ...string) error {
	return c.send(ctl.Request{Verb: verb, Units: units, Quiet: true})
}

func (c Control) Active(unit string) bool {
	response, err := ctl.Send(c.Socket, ctl.Request{Verb: "is-active", Quiet: true, Units: []string{unit}})

	return err == nil && response.Exit == 0
}

func (c Control) Property(unit, property string) string {
	response, err := ctl.Send(c.Socket, ctl.Request{Verb: "show", Units: []string{unit}, Properties: []string{property}, Value: true})
	if err != nil {
		return ""
	}

	return strings.TrimSpace(response.Stdout)
}

func (c Control) Properties(unit string, names ...string) ([]string, error) {
	response, err := ctl.Send(c.Socket, ctl.Request{Verb: "show", Units: []string{unit}, Properties: names, Value: true})
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSuffix(response.Stdout, "\n"), "\n"), nil
}

func (c Control) Run(transient ctl.Transient) error {
	return c.send(ctl.Request{Verb: "run", Quiet: true, Transient: &transient})
}

func (c Control) send(request ctl.Request) error {
	response, err := ctl.Send(c.Socket, request)
	if err != nil {
		return err
	}
	if response.Exit != 0 {
		return fmt.Errorf("ctl %s %s: %s", request.Verb, strings.Join(request.Units, " "), strings.TrimSpace(response.Stderr+response.Stdout))
	}

	return nil
}

func (c Control) Kill(signal string, units ...string) error {
	return c.send(ctl.Request{Verb: "kill", Units: units, Quiet: true, Signal: signal})
}
