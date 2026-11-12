package command

import (
	"errors"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/layout"
	"github.com/databasifylabs/engine-sdk/responses"
	"github.com/databasifylabs/engine-sdk/units"
)

type serviceControl interface {
	Do(verb string, units ...string) error
	Properties(unit string, names ...string) ([]string, error)
}

var serviceRuntime = func() serviceControl {
	return units.Control{Socket: layout.FromEnv().Socket()}
}

func controlServices(control serviceControl, verb string, names []string) (dtos.Empty, error) {
	return dtos.Empty{}, control.Do(verb, names...)
}

func reloadServices(control serviceControl) (dtos.Empty, error) {
	return dtos.Empty{}, control.Do("daemon-reload")
}

func serviceStatusOf(control serviceControl, names []string) (responses.ServiceStatus, error) {
	status := responses.ServiceStatus{Units: []responses.ServiceUnit{}}
	for _, name := range names {
		unit, err := serviceUnitOf(control, name)
		if err != nil {
			return responses.ServiceStatus{}, err
		}
		status.Units = append(status.Units, unit)
	}

	return status, nil
}

func serviceUnitOf(control serviceControl, name string) (responses.ServiceUnit, error) {
	values, err := control.Properties(name, "ActiveState", "SubState", "Result")
	if err != nil || len(values) != 3 {
		return responses.ServiceUnit{}, errors.Join(err, errors.New("the supervisor gave no status for "+name))
	}
	unit := responses.ServiceUnit{Name: name, ActiveState: values[0], SubState: values[1], Result: values[2]}
	if unit.ActiveState == "" {
		unit.ActiveState, unit.SubState = "inactive", "dead"
	}

	return unit, nil
}
