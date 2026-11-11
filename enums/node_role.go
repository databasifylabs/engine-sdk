package enums

type NodeRole string

const (
	NodeRolePrimary NodeRole = "primary"
	NodeRoleReplica NodeRole = "replica"
	NodeRoleOffline NodeRole = "offline"
	NodeRoleUnknown NodeRole = "unknown"
)

func (r NodeRole) Valid() bool {
	switch r {
	case NodeRolePrimary, NodeRoleReplica, NodeRoleOffline, NodeRoleUnknown:
		return true
	}
	return false
}
