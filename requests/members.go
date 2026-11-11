package requests

import (
	"errors"
	"fmt"
	"slices"

	"github.com/databasifylabs/engine-sdk/dtos"
)

const (
	maxVotingMembers   = 3
	minStandbyMemoryMb = 2048 * 80 / 100
)

func checkMembers(members []dtos.Member) error {
	voting := slices.DeleteFunc(slices.Clone(members), func(member dtos.Member) bool { return !member.Voting })
	if len(voting) == 0 || len(voting) > maxVotingMembers {
		return fmt.Errorf("a database has one primary and at most two standby nodes, so 1 to %d voting members; got %d", maxVotingMembers, len(voting))
	}
	for _, member := range voting {
		if len(voting) > 1 && member.MemoryMb < minStandbyMemoryMb {
			return fmt.Errorf("%s has %d MB; replication needs a 2 GB server or larger on every node of a database with standbys", member.Ip, member.MemoryMb)
		}
	}
	for _, member := range members {
		if member.Voting {
			continue
		}
		if err := readOnlyFits(member, voting); err != nil {
			return err
		}
	}

	return nil
}

func readOnlyFits(readOnly dtos.Member, voting []dtos.Member) error {
	for _, peer := range voting {
		if readOnly.Cpus < peer.Cpus || readOnly.MemoryMb < peer.MemoryMb || readOnly.DiskGb < peer.DiskGb {
			return fmt.Errorf("read-only node %s (%d CPUs, %d MB, %d GB disk) is smaller than %s (%d CPUs, %d MB, %d GB disk); a read-only node must be the same size or larger", readOnly.Ip, readOnly.Cpus, readOnly.MemoryMb, readOnly.DiskGb, peer.Ip, peer.Cpus, peer.MemoryMb, peer.DiskGb)
		}
	}

	return nil
}

func (r NodeConfigure) Check() error {
	if err := checkMembers(r.Members); err != nil {
		return err
	}
	index := slices.IndexFunc(r.Members, func(member dtos.Member) bool { return member.Ip == r.Ip })
	if index < 0 {
		return errors.New("members must list this node's ip")
	}
	if r.Members[index].Voting == r.ReadOnly {
		return errors.New("a read-only node is a non-voting member and every other node is a voting member")
	}
	if !r.Clustered && len(r.Members) != 1 {
		return errors.New("a node outside a cluster is its only member")
	}

	return nil
}

func (r ClusterMembership) Check() error {
	return checkMembers(r.Members)
}

func (r NodeConfigure) Self() dtos.Member {
	index := slices.IndexFunc(r.Members, func(member dtos.Member) bool { return member.Ip == r.Ip })
	if index < 0 {
		return dtos.Member{Ip: r.Ip, Cpus: 1, MemoryMb: 1024, DiskGb: 10}
	}

	return r.Members[index]
}

func (r NodeConfigure) Seed() bool {
	if !r.Clustered {
		return true
	}

	return !r.ReadOnly && (r.Primary == nil || *r.Primary == r.Ip)
}
