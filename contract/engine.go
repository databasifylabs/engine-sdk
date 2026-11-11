package contract

import (
	"context"
	"errors"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

var ErrUnsupported = errors.New("this engine does not implement the command")

type Report func(phase string)

type Capabilities struct {
	PasswordEncryptions       []string
	DefaultPasswordEncryption *string
	ReservedUsers             []string
	ReservedDatabases         []string
	Ports                     dtos.Ports
}

type HostProfile struct {
	Settings                    map[string]string
	Floors                      map[string]int64
	DisableTransparentHugepages bool
}

type Engine interface {
	Name() string
	Capabilities() Capabilities
	HostProfile() HostProfile
	Nodes() Nodes
	Accounts() Accounts
	Cluster() Cluster
	Archive() Archive
	Migration() Migration
	Pools() Pools
	Config() Config
}

type Config interface {
	Fields(ctx context.Context) (responses.ConfigFields, error)
	Show(ctx context.Context) (responses.ConfigShow, error)
	Tune(ctx context.Context) (responses.ConfigTune, error)
}

type Pools interface {
	List(ctx context.Context) (responses.PoolList, error)
	Create(ctx context.Context, request requests.PoolCreate) error
	Update(ctx context.Context, request requests.PoolUpdate) error
	Delete(ctx context.Context, request requests.PoolDelete) error
}

type Nodes interface {
	Status(ctx context.Context) (responses.NodeStatus, error)
	Configure(ctx context.Context, request requests.NodeConfigure) (responses.NodeConfigure, error)
	Start(ctx context.Context) error
	Restart(ctx context.Context) error
	Wake(ctx context.Context, request requests.NodeWake) error
	Ready(ctx context.Context, request requests.NodeReady, report Report) error
	Bootstrap(ctx context.Context, request requests.NodeBootstrap) error
	Metrics(ctx context.Context) (responses.NodeMetrics, error)
	ApplyConfig(ctx context.Context) error
	SyncConfig(ctx context.Context) (responses.ConfigSync, error)
	ReloadTLS(ctx context.Context) error
	InstallTLS(ctx context.Context, request requests.TlsInstall) error
	Reset(ctx context.Context, request requests.NodeReset) error
	Insights(ctx context.Context) (responses.QueryInsights, error)
}

type Accounts interface {
	List(ctx context.Context) (responses.AccountList, error)
	CreateDatabase(ctx context.Context, request requests.DatabaseCreate) (responses.DatabaseCreate, error)
	DropDatabase(ctx context.Context, request requests.DatabaseDrop) (responses.DatabaseDrop, error)
	CreateUser(ctx context.Context, request requests.UserCreate) (responses.UserCreate, error)
	SetPassword(ctx context.Context, request requests.UserSetPassword) error
	Grant(ctx context.Context, request requests.UserGrant) error
	DropUser(ctx context.Context, request requests.UserDrop) (responses.UserDrop, error)
}

type Cluster interface {
	Enable(ctx context.Context, request requests.ClusterEnable, report Report) error
	Disable(ctx context.Context) (responses.ClusterDisable, error)
	Failover(ctx context.Context, request requests.ClusterFailover) error
	ForceQuorum(ctx context.Context, request requests.ClusterForceQuorum) (responses.ClusterForceQuorum, error)
	Quorum(ctx context.Context) (responses.ClusterQuorum, error)
	Position(ctx context.Context) (responses.ClusterPosition, error)
	Admit(ctx context.Context, request requests.MemberAdmit) error
	Promote(ctx context.Context, request requests.MemberPromote) error
	Forget(ctx context.Context, request requests.MemberForget) error
	Rejoin(ctx context.Context, request requests.ClusterRejoin, report Report) error
	UpdateMembership(ctx context.Context, request requests.ClusterMembership) error
	Watch(ctx context.Context) error
}

type Archive interface {
	Status(ctx context.Context, request requests.ArchiveStatus) (responses.ArchiveStatus, error)
	Restore(ctx context.Context, request requests.RestoreStart, report Report) error
	Backup(ctx context.Context) error
	Stream(ctx context.Context) error
	Ship(ctx context.Context) error
}

type Migration interface {
	Check(ctx context.Context, request requests.MigrationCheck) (responses.MigrationCheck, error)
	Start(ctx context.Context, request requests.MigrationStart) error
	Progress(ctx context.Context) (responses.MigrationProgress, error)
	Stop(ctx context.Context) error
	Copy(ctx context.Context) error
}

type Unsupported interface {
	unsupported()
}

type Partial interface {
	Implements() []string
}
