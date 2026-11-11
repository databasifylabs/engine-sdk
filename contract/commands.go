package contract

import "github.com/databasifylabs/engine-sdk/enums"

type CommandSpec struct {
	Name  string
	Group string
	Verb  string
	Kind  enums.Kind
}

var Commands = []CommandSpec{
	{Name: "engine info", Group: "engine", Verb: "info", Kind: enums.KindQuick},
	{Name: "engine capabilities", Group: "engine", Verb: "capabilities", Kind: enums.KindQuick},
	{Name: "engine host-profile", Group: "engine", Verb: "host-profile", Kind: enums.KindQuick},
	{Name: "node status", Group: "node", Verb: "status", Kind: enums.KindQuick},
	{Name: "account list", Group: "account", Verb: "list", Kind: enums.KindQuick},
	{Name: "database create", Group: "database", Verb: "create", Kind: enums.KindQuick},
	{Name: "database drop", Group: "database", Verb: "drop", Kind: enums.KindQuick},
	{Name: "user create", Group: "user", Verb: "create", Kind: enums.KindQuick},
	{Name: "user set-password", Group: "user", Verb: "set-password", Kind: enums.KindQuick},
	{Name: "user drop", Group: "user", Verb: "drop", Kind: enums.KindQuick},
	{Name: "user grant", Group: "user", Verb: "grant", Kind: enums.KindQuick},
	{Name: "operation status", Group: "operation", Verb: "status", Kind: enums.KindQuick},
	{Name: "operation cancel", Group: "operation", Verb: "cancel", Kind: enums.KindQuick},
	{Name: "logs forward", Group: "logs", Verb: "forward", Kind: enums.KindQuick},
	{Name: "logs read", Group: "logs", Verb: "read", Kind: enums.KindQuick},
	{Name: "node configure", Group: "node", Verb: "configure", Kind: enums.KindQuick},
	{Name: "node start", Group: "node", Verb: "start", Kind: enums.KindQuick},
	{Name: "node restart", Group: "node", Verb: "restart", Kind: enums.KindQuick},
	{Name: "node wake", Group: "node", Verb: "wake", Kind: enums.KindQuick},
	{Name: "node ready", Group: "node", Verb: "ready", Kind: enums.KindLong},
	{Name: "node bootstrap", Group: "node", Verb: "bootstrap", Kind: enums.KindQuick},
	{Name: "node metrics", Group: "node", Verb: "metrics", Kind: enums.KindQuick},
	{Name: "config apply", Group: "config", Verb: "apply", Kind: enums.KindQuick},
	{Name: "config sync", Group: "config", Verb: "sync", Kind: enums.KindQuick},
	{Name: "tls reload", Group: "tls", Verb: "reload", Kind: enums.KindQuick},
	{Name: "tls install", Group: "tls", Verb: "install", Kind: enums.KindQuick},
	{Name: "cluster enable", Group: "cluster", Verb: "enable", Kind: enums.KindLong},
	{Name: "cluster disable", Group: "cluster", Verb: "disable", Kind: enums.KindQuick},
	{Name: "cluster failover", Group: "cluster", Verb: "failover", Kind: enums.KindQuick},
	{Name: "cluster force-quorum", Group: "cluster", Verb: "force-quorum", Kind: enums.KindQuick},
	{Name: "cluster quorum", Group: "cluster", Verb: "quorum", Kind: enums.KindQuick},
	{Name: "cluster position", Group: "cluster", Verb: "position", Kind: enums.KindQuick},
	{Name: "member admit", Group: "member", Verb: "admit", Kind: enums.KindQuick},
	{Name: "member promote", Group: "member", Verb: "promote", Kind: enums.KindQuick},
	{Name: "member forget", Group: "member", Verb: "forget", Kind: enums.KindQuick},
	{Name: "archive status", Group: "archive", Verb: "status", Kind: enums.KindQuick},
	{Name: "restore start", Group: "restore", Verb: "start", Kind: enums.KindLong},
	{Name: "migration check", Group: "migration", Verb: "check", Kind: enums.KindQuick},
	{Name: "migration start", Group: "migration", Verb: "start", Kind: enums.KindQuick},
	{Name: "migration progress", Group: "migration", Verb: "progress", Kind: enums.KindQuick},
	{Name: "migration stop", Group: "migration", Verb: "stop", Kind: enums.KindQuick},
	{Name: "node reset", Group: "node", Verb: "reset", Kind: enums.KindQuick},
	{Name: "query insights", Group: "query", Verb: "insights", Kind: enums.KindQuick},
	{Name: "cluster rejoin", Group: "cluster", Verb: "rejoin", Kind: enums.KindLong},
	{Name: "cluster membership", Group: "cluster", Verb: "membership", Kind: enums.KindQuick},
	{Name: "pool list", Group: "pool", Verb: "list", Kind: enums.KindQuick},
	{Name: "pool create", Group: "pool", Verb: "create", Kind: enums.KindQuick},
	{Name: "pool update", Group: "pool", Verb: "update", Kind: enums.KindQuick},
	{Name: "pool delete", Group: "pool", Verb: "delete", Kind: enums.KindQuick},
	{Name: "config fields", Group: "config", Verb: "fields", Kind: enums.KindQuick},
	{Name: "config show", Group: "config", Verb: "show", Kind: enums.KindQuick},
	{Name: "config tune", Group: "config", Verb: "tune", Kind: enums.KindQuick},
}
