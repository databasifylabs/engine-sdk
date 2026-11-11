package command

import (
	"context"
	"slices"

	"github.com/databasifylabs/engine-sdk/contract"
	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

type Engine = contract.Engine

type group func(Engine) any

func (r route) offeredBy(engine Engine, name string) bool {
	selected := r.supports(engine)
	if _, unsupported := selected.(contract.Unsupported); unsupported {
		return false
	}
	partial, isPartial := selected.(contract.Partial)

	return !isPartial || slices.Contains(partial.Implements(), name)
}

type route struct {
	supports group
	key      func(input []byte) (string, error)
	call     func(ctx context.Context, engine Engine, input []byte) ([]byte, error)
	run      func(ctx context.Context, engine Engine, input []byte, report contract.Report) error
}

func (r route) long() bool {
	return r.run != nil
}

var (
	nodes     group = func(e Engine) any { return e.Nodes() }
	accounts  group = func(e Engine) any { return e.Accounts() }
	cluster   group = func(e Engine) any { return e.Cluster() }
	archive   group = func(e Engine) any { return e.Archive() }
	migration group = func(e Engine) any { return e.Migration() }
	pools     group = func(e Engine) any { return e.Pools() }
	config    group = func(e Engine) any { return e.Config() }
)

func ask[Req, Rep any](supports group, method func(Engine) func(context.Context, Req) (Rep, error)) route {
	return route{supports: supports, call: func(ctx context.Context, engine Engine, input []byte) ([]byte, error) {
		request, err := Decode[Req](input)
		if err != nil {
			return nil, err
		}

		return Encode(method(engine)(ctx, request))
	}}
}

func tell[Req any](supports group, method func(Engine) func(context.Context, Req) error) route {
	return ask(supports, func(engine Engine) func(context.Context, Req) (dtos.Empty, error) {
		return func(ctx context.Context, request Req) (dtos.Empty, error) {
			return dtos.Empty{}, method(engine)(ctx, request)
		}
	})
}

func read[Rep any](supports group, method func(Engine) func(context.Context) (Rep, error)) route {
	return ask(supports, func(engine Engine) func(context.Context, dtos.Empty) (Rep, error) {
		return func(ctx context.Context, _ dtos.Empty) (Rep, error) {
			return method(engine)(ctx)
		}
	})
}

func act(supports group, method func(Engine) func(context.Context) error) route {
	return read(supports, func(engine Engine) func(context.Context) (dtos.Empty, error) {
		return func(ctx context.Context) (dtos.Empty, error) {
			return dtos.Empty{}, method(engine)(ctx)
		}
	})
}

func long[Req any](supports group, keyOf func(Req) string, method func(Engine) func(context.Context, Req, contract.Report) error) route {
	return route{
		supports: supports,
		key: func(input []byte) (string, error) {
			request, err := Decode[Req](input)

			return keyOf(request), err
		},
		run: func(ctx context.Context, engine Engine, input []byte, report contract.Report) error {
			request, err := Decode[Req](input)
			if err != nil {
				return err
			}

			return method(engine)(ctx, request, report)
		},
	}
}

var routes = map[string]route{
	"node status": read(nodes, func(e Engine) func(context.Context) (responses.NodeStatus, error) { return e.Nodes().Status }),
	"node configure": ask(nodes, func(e Engine) func(context.Context, requests.NodeConfigure) (responses.NodeConfigure, error) {
		return e.Nodes().Configure
	}),
	"node start":   act(nodes, func(e Engine) func(context.Context) error { return e.Nodes().Start }),
	"node restart": act(nodes, func(e Engine) func(context.Context) error { return e.Nodes().Restart }),
	"node wake":    tell(nodes, func(e Engine) func(context.Context, requests.NodeWake) error { return e.Nodes().Wake }),
	"node ready": long(nodes, func(r requests.NodeReady) string { return r.Key }, func(e Engine) func(context.Context, requests.NodeReady, contract.Report) error {
		return e.Nodes().Ready
	}),
	"node bootstrap": tell(nodes, func(e Engine) func(context.Context, requests.NodeBootstrap) error { return e.Nodes().Bootstrap }),
	"node metrics":   read(nodes, func(e Engine) func(context.Context) (responses.NodeMetrics, error) { return e.Nodes().Metrics }),
	"config apply":   act(nodes, func(e Engine) func(context.Context) error { return e.Nodes().ApplyConfig }),
	"config sync":    read(nodes, func(e Engine) func(context.Context) (responses.ConfigSync, error) { return e.Nodes().SyncConfig }),
	"tls reload":     act(nodes, func(e Engine) func(context.Context) error { return e.Nodes().ReloadTLS }),
	"tls install": tell(nodes, func(e Engine) func(context.Context, requests.TlsInstall) error {
		return e.Nodes().InstallTLS
	}),
	"node reset":     tell(nodes, func(e Engine) func(context.Context, requests.NodeReset) error { return e.Nodes().Reset }),
	"query insights": read(nodes, func(e Engine) func(context.Context) (responses.QueryInsights, error) { return e.Nodes().Insights }),

	"config fields": read(config, func(e Engine) func(context.Context) (responses.ConfigFields, error) { return e.Config().Fields }),
	"config show":   read(config, func(e Engine) func(context.Context) (responses.ConfigShow, error) { return e.Config().Show }),
	"config tune":   read(config, func(e Engine) func(context.Context) (responses.ConfigTune, error) { return e.Config().Tune }),
	"pool list":     read(pools, func(e Engine) func(context.Context) (responses.PoolList, error) { return e.Pools().List }),
	"pool create":   tell(pools, func(e Engine) func(context.Context, requests.PoolCreate) error { return e.Pools().Create }),
	"pool update":   tell(pools, func(e Engine) func(context.Context, requests.PoolUpdate) error { return e.Pools().Update }),
	"pool delete":   tell(pools, func(e Engine) func(context.Context, requests.PoolDelete) error { return e.Pools().Delete }),
	"account list":  read(accounts, func(e Engine) func(context.Context) (responses.AccountList, error) { return e.Accounts().List }),
	"database create": ask(accounts, func(e Engine) func(context.Context, requests.DatabaseCreate) (responses.DatabaseCreate, error) {
		return e.Accounts().CreateDatabase
	}),
	"database drop": ask(accounts, func(e Engine) func(context.Context, requests.DatabaseDrop) (responses.DatabaseDrop, error) {
		return e.Accounts().DropDatabase
	}),
	"user create": ask(accounts, func(e Engine) func(context.Context, requests.UserCreate) (responses.UserCreate, error) {
		return e.Accounts().CreateUser
	}),
	"user set-password": tell(accounts, func(e Engine) func(context.Context, requests.UserSetPassword) error {
		return e.Accounts().SetPassword
	}),
	"user grant": tell(accounts, func(e Engine) func(context.Context, requests.UserGrant) error { return e.Accounts().Grant }),
	"user drop": ask(accounts, func(e Engine) func(context.Context, requests.UserDrop) (responses.UserDrop, error) {
		return e.Accounts().DropUser
	}),

	"cluster enable": long(cluster, func(r requests.ClusterEnable) string { return r.Key }, func(e Engine) func(context.Context, requests.ClusterEnable, contract.Report) error {
		return e.Cluster().Enable
	}),
	"cluster disable": read(cluster, func(e Engine) func(context.Context) (responses.ClusterDisable, error) { return e.Cluster().Disable }),
	"cluster failover": tell(cluster, func(e Engine) func(context.Context, requests.ClusterFailover) error {
		return e.Cluster().Failover
	}),
	"cluster force-quorum": ask(cluster, func(e Engine) func(context.Context, requests.ClusterForceQuorum) (responses.ClusterForceQuorum, error) {
		return e.Cluster().ForceQuorum
	}),
	"cluster quorum": read(cluster, func(e Engine) func(context.Context) (responses.ClusterQuorum, error) { return e.Cluster().Quorum }),
	"cluster position": read(cluster, func(e Engine) func(context.Context) (responses.ClusterPosition, error) {
		return e.Cluster().Position
	}),
	"member admit":   tell(cluster, func(e Engine) func(context.Context, requests.MemberAdmit) error { return e.Cluster().Admit }),
	"member promote": tell(cluster, func(e Engine) func(context.Context, requests.MemberPromote) error { return e.Cluster().Promote }),
	"member forget":  tell(cluster, func(e Engine) func(context.Context, requests.MemberForget) error { return e.Cluster().Forget }),
	"cluster rejoin": long(cluster, func(r requests.ClusterRejoin) string { return r.Key }, func(e Engine) func(context.Context, requests.ClusterRejoin, contract.Report) error {
		return e.Cluster().Rejoin
	}),
	"cluster membership": tell(cluster, func(e Engine) func(context.Context, requests.ClusterMembership) error {
		return e.Cluster().UpdateMembership
	}),

	"archive status": ask(archive, func(e Engine) func(context.Context, requests.ArchiveStatus) (responses.ArchiveStatus, error) {
		return e.Archive().Status
	}),
	"restore start": long(archive, func(r requests.RestoreStart) string { return r.Key }, func(e Engine) func(context.Context, requests.RestoreStart, contract.Report) error {
		return e.Archive().Restore
	}),

	"migration check": ask(migration, func(e Engine) func(context.Context, requests.MigrationCheck) (responses.MigrationCheck, error) {
		return e.Migration().Check
	}),
	"migration start": tell(migration, func(e Engine) func(context.Context, requests.MigrationStart) error { return e.Migration().Start }),
	"migration progress": read(migration, func(e Engine) func(context.Context) (responses.MigrationProgress, error) {
		return e.Migration().Progress
	}),
	"migration stop": act(migration, func(e Engine) func(context.Context) error { return e.Migration().Stop }),
}

var tasks = map[string]func(Engine) func(context.Context) error{
	"backup":         func(e Engine) func(context.Context) error { return e.Archive().Backup },
	"archive-stream": func(e Engine) func(context.Context) error { return e.Archive().Stream },
	"archive-ship":   func(e Engine) func(context.Context) error { return e.Archive().Ship },
	"migration-copy": func(e Engine) func(context.Context) error { return e.Migration().Copy },
	"cluster-watch":  func(e Engine) func(context.Context) error { return e.Cluster().Watch },
}

func RunTask(ctx context.Context, engine Engine, name string) error {
	task, known := tasks[name]
	if !known {
		return Unsupported(name)
	}

	return task(engine)(ctx)
}
