package host

import (
	"strings"
	"testing"
)

type listedContainers string

func (l listedContainers) Run(...string) (string, error) {
	return string(l), nil
}

func (listedContainers) RunWithInput(string, ...string) (string, error) {
	return "", nil
}

func TestEngineContainerIsTheOneRunningEngine(t *testing.T) {
	name, err := engineContainer(listedContainers("databasify-postgres\n"))

	if err != nil || name != "databasify-postgres" {
		t.Fatalf("name = %q, err = %v", name, err)
	}
}

func TestEngineContainerRefusesToGuessBetweenTwoEngines(t *testing.T) {
	_, err := engineContainer(listedContainers("databasify-postgres\ndatabasify-valkey\n"))

	if err == nil || !strings.Contains(err.Error(), "more than one engine container is running") {
		t.Fatalf("err = %v", err)
	}
}

type labelledContainers map[string]string

func (l labelledContainers) Run(args ...string) (string, error) {
	return l[strings.Join(args[:3], " ")], nil
}

func (labelledContainers) RunWithInput(string, ...string) (string, error) {
	return "", nil
}

func TestEngineContainerIsFoundByTheEngineLabelAlone(t *testing.T) {
	docker := labelledContainers{"ps --filter label=io.databasify.engine": "databasify-postgres\n"}

	name, err := engineContainer(docker)

	if err != nil || name != "databasify-postgres" {
		t.Fatalf("name = %q, err = %v", name, err)
	}
}
