package command

import (
	"errors"
	"slices"
	"testing"
)

type recordingControl struct {
	sent       []string
	properties map[string]string
}

func (r *recordingControl) Do(verb string, units ...string) error {
	r.sent = append(r.sent, verb+" "+joined(units))

	return nil
}

func (r *recordingControl) Properties(unit string, names ...string) ([]string, error) {
	values := []string{}
	for _, name := range names {
		values = append(values, r.properties[unit+"."+name])
	}

	return values, nil
}

func joined(units []string) string {
	text := ""
	for _, unit := range units {
		text += unit + ","
	}

	return text
}

func TestServiceRestartSendsTheVerbForTheUnits(t *testing.T) {
	control := &recordingControl{}

	_, err := controlServices(control, "restart", []string{"postgres"})

	if err != nil || !slices.Equal(control.sent, []string{"restart postgres,"}) {
		t.Fatalf("%v %v", control.sent, err)
	}
}

func TestServiceReloadReadsTheManifestAgain(t *testing.T) {
	control := &recordingControl{}

	_, err := reloadServices(control)

	if err != nil || !slices.Equal(control.sent, []string{"daemon-reload "}) {
		t.Fatalf("%v %v", control.sent, err)
	}
}

func TestServiceStatusReportsAnUnknownUnitAsInactive(t *testing.T) {
	control := &recordingControl{properties: map[string]string{"postgres.ActiveState": "active", "postgres.SubState": "running", "postgres.Result": "success"}}

	status, err := serviceStatusOf(control, []string{"postgres", "ghost"})

	if err != nil || len(status.Units) != 2 || status.Units[0].ActiveState != "active" || status.Units[0].SubState != "running" || status.Units[1].ActiveState != "inactive" || status.Units[1].Name != "ghost" {
		t.Fatalf("%+v", status)
	}
}

type unreachableControl struct{}

func (unreachableControl) Do(string, ...string) error {
	return errors.New("dial unix: connection refused")
}

func (unreachableControl) Properties(string, ...string) ([]string, error) {
	return nil, errors.New("dial unix: connection refused")
}

func TestServiceStatusFailsWhenTheSupervisorIsUnreachable(t *testing.T) {
	if _, err := serviceStatusOf(unreachableControl{}, []string{"postgres"}); err == nil {
		t.Fatal("an unreachable supervisor was reported as a status")
	}
}
