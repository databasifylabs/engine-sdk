package contract

import (
	"context"

	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

type UnsupportedNodes struct{}

func (UnsupportedNodes) unsupported() {}

func (UnsupportedNodes) Status(context.Context) (responses.NodeStatus, error) {
	return responses.NodeStatus{}, ErrUnsupported
}

func (UnsupportedNodes) Configure(context.Context, requests.NodeConfigure) (responses.NodeConfigure, error) {
	return responses.NodeConfigure{}, ErrUnsupported
}

func (UnsupportedNodes) Start(context.Context) error { return ErrUnsupported }

func (UnsupportedNodes) Restart(context.Context) error { return ErrUnsupported }

func (UnsupportedNodes) Wake(context.Context, requests.NodeWake) error { return ErrUnsupported }

func (UnsupportedNodes) Ready(context.Context, requests.NodeReady, Report) error {
	return ErrUnsupported
}

func (UnsupportedNodes) Bootstrap(context.Context, requests.NodeBootstrap) error {
	return ErrUnsupported
}

func (UnsupportedNodes) Metrics(context.Context) (responses.NodeMetrics, error) {
	return responses.NodeMetrics{}, ErrUnsupported
}

func (UnsupportedNodes) ApplyConfig(context.Context) error { return ErrUnsupported }

func (UnsupportedNodes) SyncConfig(context.Context) (responses.ConfigSync, error) {
	return responses.ConfigSync{}, ErrUnsupported
}

func (UnsupportedNodes) ReloadTLS(context.Context) error { return ErrUnsupported }

func (UnsupportedNodes) InstallTLS(context.Context, requests.TlsInstall) error {
	return ErrUnsupported
}

func (UnsupportedNodes) Reset(context.Context, requests.NodeReset) error { return ErrUnsupported }

func (UnsupportedNodes) Insights(context.Context) (responses.QueryInsights, error) {
	return responses.QueryInsights{}, ErrUnsupported
}

type UnsupportedAccounts struct{}

func (UnsupportedAccounts) unsupported() {}

func (UnsupportedAccounts) List(context.Context) (responses.AccountList, error) {
	return responses.AccountList{}, ErrUnsupported
}

func (UnsupportedAccounts) CreateDatabase(context.Context, requests.DatabaseCreate) (responses.DatabaseCreate, error) {
	return responses.DatabaseCreate{}, ErrUnsupported
}

func (UnsupportedAccounts) DropDatabase(context.Context, requests.DatabaseDrop) (responses.DatabaseDrop, error) {
	return responses.DatabaseDrop{}, ErrUnsupported
}

func (UnsupportedAccounts) CreateUser(context.Context, requests.UserCreate) (responses.UserCreate, error) {
	return responses.UserCreate{}, ErrUnsupported
}

func (UnsupportedAccounts) SetPassword(context.Context, requests.UserSetPassword) error {
	return ErrUnsupported
}

func (UnsupportedAccounts) Grant(context.Context, requests.UserGrant) error { return ErrUnsupported }

func (UnsupportedAccounts) DropUser(context.Context, requests.UserDrop) (responses.UserDrop, error) {
	return responses.UserDrop{}, ErrUnsupported
}

type UnsupportedCluster struct{}

func (UnsupportedCluster) unsupported() {}

func (UnsupportedCluster) Enable(context.Context, requests.ClusterEnable, Report) error {
	return ErrUnsupported
}

func (UnsupportedCluster) Disable(context.Context) (responses.ClusterDisable, error) {
	return responses.ClusterDisable{}, ErrUnsupported
}

func (UnsupportedCluster) Failover(context.Context, requests.ClusterFailover) error {
	return ErrUnsupported
}

func (UnsupportedCluster) ForceQuorum(context.Context, requests.ClusterForceQuorum) (responses.ClusterForceQuorum, error) {
	return responses.ClusterForceQuorum{}, ErrUnsupported
}

func (UnsupportedCluster) Quorum(context.Context) (responses.ClusterQuorum, error) {
	return responses.ClusterQuorum{}, ErrUnsupported
}

func (UnsupportedCluster) Position(context.Context) (responses.ClusterPosition, error) {
	return responses.ClusterPosition{}, ErrUnsupported
}

func (UnsupportedCluster) Admit(context.Context, requests.MemberAdmit) error { return ErrUnsupported }

func (UnsupportedCluster) Promote(context.Context, requests.MemberPromote) error {
	return ErrUnsupported
}

func (UnsupportedCluster) Forget(context.Context, requests.MemberForget) error { return ErrUnsupported }

func (UnsupportedCluster) Rejoin(context.Context, requests.ClusterRejoin, Report) error {
	return ErrUnsupported
}

func (UnsupportedCluster) UpdateMembership(context.Context, requests.ClusterMembership) error {
	return ErrUnsupported
}

type UnsupportedArchive struct{}

func (UnsupportedArchive) unsupported() {}

func (UnsupportedArchive) Status(context.Context, requests.ArchiveStatus) (responses.ArchiveStatus, error) {
	return responses.ArchiveStatus{}, ErrUnsupported
}

func (UnsupportedArchive) Restore(context.Context, requests.RestoreStart, Report) error {
	return ErrUnsupported
}

func (UnsupportedArchive) Backup(context.Context) error { return ErrUnsupported }

func (UnsupportedArchive) Stream(context.Context) error { return ErrUnsupported }

func (UnsupportedArchive) Ship(context.Context) error { return ErrUnsupported }

type UnsupportedMigration struct{}

func (UnsupportedMigration) unsupported() {}

func (UnsupportedMigration) Check(context.Context, requests.MigrationCheck) (responses.MigrationCheck, error) {
	return responses.MigrationCheck{}, ErrUnsupported
}

func (UnsupportedMigration) Start(context.Context, requests.MigrationStart) error {
	return ErrUnsupported
}

func (UnsupportedMigration) Progress(context.Context) (responses.MigrationProgress, error) {
	return responses.MigrationProgress{}, ErrUnsupported
}

func (UnsupportedMigration) Stop(context.Context) error { return ErrUnsupported }

func (UnsupportedMigration) Copy(context.Context) error { return ErrUnsupported }

func (UnsupportedCluster) Watch(context.Context) error { return ErrUnsupported }

type UnsupportedPools struct{}

func (UnsupportedPools) unsupported() {}

func (UnsupportedPools) List(context.Context) (responses.PoolList, error) {
	return responses.PoolList{}, ErrUnsupported
}

func (UnsupportedPools) Create(context.Context, requests.PoolCreate) error { return ErrUnsupported }

func (UnsupportedPools) Update(context.Context, requests.PoolUpdate) error { return ErrUnsupported }

func (UnsupportedPools) Delete(context.Context, requests.PoolDelete) error { return ErrUnsupported }

type UnsupportedConfig struct{}

func (UnsupportedConfig) unsupported() {}

func (UnsupportedConfig) Fields(context.Context) (responses.ConfigFields, error) {
	return responses.ConfigFields{}, ErrUnsupported
}

func (UnsupportedConfig) Show(context.Context) (responses.ConfigShow, error) {
	return responses.ConfigShow{}, ErrUnsupported
}

func (UnsupportedConfig) Tune(context.Context) (responses.ConfigTune, error) {
	return responses.ConfigTune{}, ErrUnsupported
}
