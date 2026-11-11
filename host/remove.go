package host

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/databasifylabs/engine-sdk/nodefs"
	"github.com/databasifylabs/engine-sdk/requests"
)

func remove(docker Docker, srv string, restoreHost func(engine string) error, request requests.HostRemove) error {
	engine, found := engineHoldingIn(srv, request.DatabaseId)
	if !found {
		return nil
	}
	container := containerOf(engine)
	if request.Erase {
		if err := restoreHost(engine); err != nil {
			return err
		}
	}
	_, _ = docker.Run("stop", "--time", stopTimeout, container)
	if !request.Erase {
		return nil
	}
	_, _ = docker.Run("rm", "-v", "-f", container)
	removeImagesOf(docker, engine)

	return os.RemoveAll(filepath.Join(srv, engine))
}

func engineHoldingIn(srv string, databaseId string) (string, bool) {
	entries, err := os.ReadDir(srv)
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		marker := strings.TrimSpace(nodefs.Root(filepath.Join(srv, entry.Name())).Read(databaseMarker))
		if entry.IsDir() && marker == databaseId {
			return entry.Name(), true
		}
	}

	return "", false
}

func removeImagesOf(docker Docker, engine string) {
	listed, err := docker.Run("image", "ls", "--quiet", "--filter", "label="+engineLabelKey+"="+engine)
	if err != nil {
		return
	}
	for _, id := range strings.Fields(listed) {
		_, _ = docker.Run("image", "rm", "-f", id)
	}
}
