package host

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/databasifylabs/engine-sdk/responses"
)

const (
	originalsDir = "etc/databasify/sysctl-originals"
	thpOriginals = "etc/databasify/thp-originals"
	thpUnit      = "etc/systemd/system/disable-thp.service"
	thpSettings  = "sys/kernel/mm/transparent_hugepage"
	sysctlDir    = "etc/sysctl.d"
	procSys      = "proc/sys"
)

const thpUnitText = `[Unit]
Description=Disable transparent huge pages for the database
After=tuned.service
Before=docker.service

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'echo never > /sys/kernel/mm/transparent_hugepage/enabled && echo never > /sys/kernel/mm/transparent_hugepage/defrag'

[Install]
WantedBy=multi-user.target
`

type system struct {
	root string
	run  func(name string, args ...string) error
}

var hostSystem = system{root: "/", run: run}

func settingsOf(profile responses.EngineHostProfile, current func(key string) int64) map[string]string {
	settings := maps.Clone(profile.Settings)
	if settings == nil {
		settings = map[string]string{}
	}
	for key, floor := range profile.Floors {
		settings[key] = strconv.FormatInt(max(current(key), floor), 10)
	}

	return settings
}

func (s system) path(relative string) string {
	return filepath.Join(s.root, relative)
}

func (s system) tune(engine string, profile responses.EngineHostProfile) (responses.HostTune, error) {
	settings := settingsOf(profile, s.originalValue(engine))
	if len(settings) == 0 {
		return responses.HostTune{Settings: settings}, nil
	}
	if err := s.saveOriginals(engine, settings); err != nil {
		return responses.HostTune{}, err
	}
	if err := os.WriteFile(s.tuningFile(engine), []byte(sysctlLines(settings)), 0o644); err != nil {
		return responses.HostTune{}, err
	}
	if err := s.run("sysctl", "--system"); err != nil {
		return responses.HostTune{}, err
	}
	if profile.DisableTransparentHugepages {
		return responses.HostTune{Settings: settings}, s.disableTransparentHugePages()
	}

	return responses.HostTune{Settings: settings}, nil
}

func (s system) restore(engine string) error {
	originals := filepath.Join(s.path(originalsDir), engine)
	if _, err := os.Stat(originals); err == nil {
		_ = s.run("sysctl", "-q", "-p", originals)
		_ = os.Remove(originals)
	}
	_ = os.Remove(s.tuningFile(engine))
	_ = os.Remove(s.path(originalsDir))
	_ = s.run("sysctl", "--system")
	s.restoreTransparentHugePages()
	_ = os.Remove(filepath.Dir(s.path(originalsDir)))

	return nil
}

func (s system) tuningFile(engine string) string {
	return filepath.Join(s.path(sysctlDir), "99-"+engine+".conf")
}

func (s system) originalValue(engine string) func(key string) int64 {
	saved := s.savedOriginals(engine)

	return func(key string) int64 {
		if value, found := saved[key]; found {
			number, _ := strconv.ParseInt(value, 10, 64)

			return number
		}
		number, _ := strconv.ParseInt(s.liveValue(key), 10, 64)

		return number
	}
}

func (s system) liveValue(key string) string {
	data, err := os.ReadFile(filepath.Join(s.path(procSys), strings.ReplaceAll(key, ".", "/")))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func (s system) savedOriginals(engine string) map[string]string {
	saved := map[string]string{}
	data, err := os.ReadFile(filepath.Join(s.path(originalsDir), engine))
	if err != nil {
		return saved
	}
	for line := range strings.Lines(string(data)) {
		key, value, found := strings.Cut(strings.TrimSpace(line), " = ")
		if found {
			saved[key] = value
		}
	}

	return saved
}

func (s system) saveOriginals(engine string, settings map[string]string) error {
	path := filepath.Join(s.path(originalsDir), engine)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if _, err := os.Stat(s.tuningFile(engine)); err == nil {
		return nil
	}
	if err := os.MkdirAll(s.path(originalsDir), 0o755); err != nil {
		return err
	}
	originals := map[string]string{}
	for key := range settings {
		originals[key] = s.liveValue(key)
	}

	return os.WriteFile(path, []byte(sysctlLines(originals)), 0o644)
}

func sysctlLines(settings map[string]string) string {
	var lines strings.Builder
	for _, key := range slices.Sorted(maps.Keys(settings)) {
		lines.WriteString(key + " = " + settings[key] + "\n")
	}

	return lines.String()
}

func (s system) disableTransparentHugePages() error {
	if _, err := os.Stat(s.path(thpOriginals)); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(s.path(thpOriginals)), 0o755); err != nil {
			return err
		}
		lines := "enabled=" + s.selectedThp("enabled") + "\ndefrag=" + s.selectedThp("defrag") + "\n"
		if err := os.WriteFile(s.path(thpOriginals), []byte(lines), 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(s.path(thpUnit), []byte(thpUnitText), 0o644); err != nil {
		return err
	}
	if err := s.run("systemctl", "daemon-reload"); err != nil {
		return err
	}

	return s.run("systemctl", "enable", "--now", "disable-thp.service")
}

func (s system) restoreTransparentHugePages() {
	if _, err := os.Stat(s.path(thpUnit)); err != nil {
		return
	}
	_ = s.run("systemctl", "disable", "--now", "disable-thp.service")
	_ = os.Remove(s.path(thpUnit))
	_ = s.run("systemctl", "daemon-reload")
	data, _ := os.ReadFile(s.path(thpOriginals))
	for line := range strings.Lines(string(data)) {
		setting, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if found && value != "" {
			_ = os.WriteFile(filepath.Join(s.path(thpSettings), setting), []byte(value), 0o644)
		}
	}
	_ = os.Remove(s.path(thpOriginals))
}

func (s system) selectedThp(setting string) string {
	data, err := os.ReadFile(filepath.Join(s.path(thpSettings), setting))
	if err != nil {
		return "madvise"
	}
	_, rest, found := strings.Cut(string(data), "[")
	selected, _, _ := strings.Cut(rest, "]")
	if !found || selected == "" {
		return "madvise"
	}

	return selected
}

func run(name string, args ...string) error {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}

	return nil
}
