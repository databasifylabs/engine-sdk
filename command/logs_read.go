package command

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/databasifylabs/engine-sdk/ctl"
	"github.com/databasifylabs/engine-sdk/layout"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

func readLogs(paths layout.Layout, request requests.LogsRead) (responses.LogsRead, error) {
	units, err := logUnitsIn(paths.Logs())
	if err != nil {
		return responses.LogsRead{}, err
	}
	logs := responses.LogsRead{Units: servicesIn(units), Lines: []string{}}
	if request.Unit == nil {
		return logs, nil
	}
	if !slices.Contains(units, *request.Unit) {
		return responses.LogsRead{}, NotFound("no log for " + *request.Unit)
	}
	lines, err := ctl.TailOf(paths.LogFile(*request.Unit), int(request.Lines))
	if err != nil {
		return responses.LogsRead{}, err
	}
	logs.Lines = append(logs.Lines, lines...)

	return logs, nil
}

func servicesIn(units []string) []string {
	return slices.DeleteFunc(slices.Clone(units), func(name string) bool { return strings.HasPrefix(name, unitOf("")) })
}

func logUnitsIn(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	units := []string{}
	for _, entry := range entries {
		if name, isLog := strings.CutSuffix(entry.Name(), ".log"); isLog && !entry.IsDir() {
			units = append(units, name)
		}
	}
	slices.Sort(units)

	return units, nil
}
