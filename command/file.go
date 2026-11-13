package command

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

func putFile(home string, request requests.FilePut) error {
	root, target, err := rootedPath(home, request.Path)
	if err != nil {
		return err
	}
	defer root.Close()
	content, err := base64.StdEncoding.DecodeString(request.ContentBase64)
	if err != nil {
		return Invalid("contentBase64 is not base64")
	}
	if err := root.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	staging := target + ".tmp-" + rand.Text()
	if err := writeStaging(root, staging, content, request); err != nil {
		_ = root.Remove(staging)

		return err
	}

	return root.Rename(staging, target)
}

func writeStaging(root *os.Root, staging string, content []byte, request requests.FilePut) error {
	file, err := root.OpenFile(staging, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := file.Chmod(os.FileMode(request.Mode)); err != nil {
		return err
	}

	if err := chownTo(file, request.Owner); err != nil {
		return err
	}

	return file.Sync()
}

func getFile(home string, request requests.FileGet) (responses.FileGet, error) {
	root, target, err := rootedPath(home, request.Path)
	if err != nil {
		return responses.FileGet{}, err
	}
	defer root.Close()
	content, err := root.ReadFile(target)
	if errors.Is(err, os.ErrNotExist) {
		return responses.FileGet{}, nil
	}
	if err != nil {
		return responses.FileGet{}, err
	}

	return responses.FileGet{ContentBase64: base64.StdEncoding.EncodeToString(content), Exists: true}, nil
}

func rootedPath(home string, path string) (*os.Root, string, error) {
	clean := filepath.Clean(path)
	if path == "" || filepath.IsAbs(path) || clean == ".." || strings.HasPrefix(clean, "../") {
		return nil, "", Invalid("the path must stay inside the engine home: " + path)
	}
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, "", err
	}
	if resolved, err := filepath.EvalSymlinks(existingAncestorOf(filepath.Join(resolvedHome, clean))); err != nil || !within(resolvedHome, resolved) {
		return nil, "", Invalid("the path must stay inside the engine home: " + path)
	}
	root, err := os.OpenRoot(resolvedHome)

	return root, clean, err
}

func existingAncestorOf(path string) string {
	for {
		if _, err := os.Lstat(path); err == nil {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		path = parent
	}
}

func within(root string, path string) bool {
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

func chownTo(file *os.File, owner *string) error {
	if owner == nil {
		return nil
	}
	account, err := user.Lookup(*owner)
	if err != nil {
		return Invalid("no user " + *owner)
	}
	uid, _ := strconv.Atoi(account.Uid)
	gid, _ := strconv.Atoi(account.Gid)

	return file.Chown(uid, gid)
}
