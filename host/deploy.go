package host

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/nodefs"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

const (
	srvRoot        = "/srv"
	databaseMarker = ".database"
	stopTimeout    = "960"
	openFiles      = "65536"
	engineLabelKey = "io.databasify.engine"
	sdkLabelKey    = "io.databasify.sdk"
)

func containerOf(engine string) string {
	return "databasify-" + engine
}

func deploy(docker Docker, srv string, request requests.HostDeploy) (responses.HostDeploy, error) {
	if _, err := docker.Run("version", "--format", "{{.Server.Version}}"); err != nil {
		return responses.HostDeploy{}, command.Unavailable("Docker is not installed or not running: " + err.Error())
	}
	if err := pullIfMissing(docker, request.Image); err != nil {
		return responses.HostDeploy{}, err
	}
	engine, err := engineOf(docker, request.Image)
	if err != nil {
		return responses.HostDeploy{}, err
	}
	home, reply := filepath.Join(srv, engine), responses.HostDeploy{Engine: engine, Container: containerOf(engine)}
	current := containerStateOf(docker, reply.Container)
	if err := refuseAnotherDatabase(home, current, request.DatabaseId); err != nil {
		return responses.HostDeploy{}, err
	}
	if err := prepareHome(srv, engine, request.DatabaseId); err != nil {
		return responses.HostDeploy{}, err
	}
	reply.Created, reply.Replaced, err = runContainer(docker, home, reply.Container, current, request)
	if err != nil {
		return responses.HostDeploy{}, err
	}
	pruneImagesBeside(docker, request.Image)

	return reply, nil
}

func engineOf(docker Docker, image string) (string, error) {
	labels, err := docker.Run("image", "inspect", "--format", `{{index .Config.Labels "`+engineLabelKey+`"}}|{{index .Config.Labels "`+sdkLabelKey+`"}}`, image)
	if err != nil {
		return "", err
	}
	engine, sdk, _ := strings.Cut(labels, "|")
	if engine == "" {
		return "", command.Invalid(image + " is not a Databasify engine image")
	}
	if sdk == "" {
		return "", command.Invalid(image + " is not built on the Databasify engine SDK")
	}

	return engine, nil
}

func pullIfMissing(docker Docker, image string) error {
	if _, err := docker.Run("image", "inspect", image); err == nil {
		return nil
	}
	if _, err := docker.Run("pull", image); err != nil {
		return command.Unavailable("could not pull " + image + ": " + err.Error())
	}

	return nil
}

type containerState struct {
	exists   bool
	image    string
	running  bool
	database string
}

func containerStateOf(docker Docker, container string) containerState {
	state, err := docker.Run("container", "inspect", "--format", `{{.Config.Image}} {{.State.Running}} {{index .Config.Labels "io.databasify.database"}}`, container)
	if err != nil {
		return containerState{}
	}
	fields := strings.Fields(state)
	for len(fields) < 3 {
		fields = append(fields, "")
	}

	return containerState{exists: true, image: fields[0], running: fields[1] == "true", database: fields[2]}
}

func refuseAnotherDatabase(home string, current containerState, databaseId string) error {
	if current.exists && current.database != databaseId {
		return busyWith(home, current.database)
	}
	owner := strings.TrimSpace(nodefs.Root(home).Read(databaseMarker))
	if owner != "" && owner != databaseId {
		return busyWith(home, owner)
	}
	if owner == "" && !nodefs.Root(home).Empty("data") {
		return busyWith(home, "unknown")
	}

	return nil
}

func busyWith(home string, owner string) error {
	return command.Failure{Code: enums.ErrorCodeBusy, Message: home + " belongs to another database (" + owner + "); erase it with host remove before reusing this server"}
}

func prepareHome(srv string, engine string, databaseId string) error {
	root := nodefs.Root(filepath.Join(srv, engine))
	for _, dir := range []string{"", "tls", "conf"} {
		if err := root.Dir(dir); err != nil {
			return err
		}
	}
	if err := os.Chmod(root.Path(""), 0o755); err != nil {
		return err
	}
	_, err := root.Write(databaseMarker, databaseId, 0o644, true)

	return err
}

func runContainer(docker Docker, home string, container string, current containerState, request requests.HostDeploy) (bool, bool, error) {
	if !current.exists {
		_, err := docker.Run(runArgs(home, container, request)...)

		return err == nil, false, err
	}
	if current.image == request.Image {
		if current.running {
			return false, false, nil
		}
		_, err := docker.Run("start", container)

		return false, false, err
	}
	if _, err := docker.Run("stop", "--time", stopTimeout, container); err != nil {
		return false, false, err
	}
	if _, err := docker.Run("rm", "-v", container); err != nil {
		return false, false, err
	}
	_, err := docker.Run(runArgs(home, container, request)...)

	return false, err == nil, err
}

func runArgs(home string, container string, request requests.HostDeploy) []string {
	return []string{
		"run", "-d", "--name", container,
		"--network", "host", "--ipc", "host", "--security-opt", "label=disable",
		"--restart", "unless-stopped", "--stop-timeout", stopTimeout,
		"--ulimit", "nofile=" + openFiles + ":" + openFiles,
		"-v", home + ":" + home,
		"--label", "io.databasify.database=" + request.DatabaseId,
		"--label", "io.databasify.node=" + request.NodeId,
		"-e", "DATABASIFY_IMAGE=" + request.Image,
		request.Image,
	}
}

func pruneImagesBeside(docker Docker, image string) {
	repository := repositoryOf(image)
	current, err := docker.Run("image", "inspect", "--format", "{{.Id}}", image)
	if err != nil {
		return
	}
	listed, err := docker.Run("image", "ls", "--no-trunc", "--filter", "reference="+repository, "--format", "{{.ID}}")
	if err != nil {
		return
	}
	for _, id := range strings.Fields(listed) {
		if id != current {
			_, _ = docker.Run("image", "rm", id)
		}
	}
}

func repositoryOf(image string) string {
	name, _, _ := strings.Cut(image, "@")
	if slash := strings.LastIndex(name, "/"); strings.LastIndex(name, ":") > slash {
		name = name[:strings.LastIndex(name, ":")]
	}

	return name
}
