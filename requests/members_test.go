package requests

import (
	"testing"

	"github.com/databasifylabs/engine-sdk/dtos"
)

func members(voting int, readOnly int) []dtos.Member {
	list := []dtos.Member{}
	for index := range voting + readOnly {
		list = append(list, dtos.Member{Ip: "10.0.0." + string(rune('1'+index)), Position: int64(index + 1), Voting: index < voting, Cpus: 2, MemoryMb: 4096, DiskGb: 100})
	}

	return list
}

func TestAClusterHasOnePrimaryAndAtMostTwoStandbys(t *testing.T) {
	for voting, fits := range map[int]bool{0: false, 1: true, 2: true, 3: true, 4: false} {
		err := NodeConfigure{Clustered: true, Ip: "10.0.0.1", Members: members(voting, 0)}.Check()
		if (err == nil) != fits {
			t.Errorf("%d voting: err = %v", voting, err)
		}
	}
	if (ClusterMembership{Members: members(4, 0)}).Check() == nil {
		t.Fatal("membership must refuse a fourth voting member")
	}
}

func TestReadOnlyNodesAreUnlimited(t *testing.T) {
	request := NodeConfigure{Clustered: true, ReadOnly: true, Ip: "10.0.0.4", Members: members(3, 6)}

	if err := request.Check(); err != nil {
		t.Fatal(err)
	}
}

func TestTheNodeMustBeListedWithItsRole(t *testing.T) {
	cases := map[string]NodeConfigure{
		"missing":           {Clustered: true, Ip: "10.9.9.9", Members: members(3, 0)},
		"replica voting":    {Clustered: true, ReadOnly: true, Ip: "10.0.0.1", Members: members(3, 0)},
		"single with peers": {Ip: "10.0.0.1", Members: members(2, 0)},
	}
	for name, request := range cases {
		if request.Check() == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if err := (NodeConfigure{Ip: "10.0.0.1", Members: members(1, 0)}).Check(); err != nil {
		t.Fatalf("a single node: %v", err)
	}
}

func TestReadOnlyNodesAreNeverSmallerThanVotingMembers(t *testing.T) {
	list := members(2, 1)
	list[2].MemoryMb = 2048
	if (NodeConfigure{Clustered: true, ReadOnly: true, Ip: list[2].Ip, Members: list}).Check() == nil {
		t.Fatal("a read-only node with less memory than the primary was accepted")
	}
	list[2].MemoryMb, list[2].Cpus = 8192, 1
	if (NodeConfigure{Clustered: true, ReadOnly: true, Ip: list[2].Ip, Members: list}).Check() == nil {
		t.Fatal("a read-only node with fewer CPUs than the primary was accepted")
	}
	list[2].Cpus, list[2].DiskGb = 4, 50
	if (NodeConfigure{Clustered: true, ReadOnly: true, Ip: list[2].Ip, Members: list}).Check() == nil {
		t.Fatal("a read-only node with a smaller disk than the primary was accepted")
	}
	list[2].DiskGb = 200
	if err := (NodeConfigure{Clustered: true, ReadOnly: true, Ip: list[2].Ip, Members: list}).Check(); err != nil {
		t.Fatalf("a larger read-only node was refused: %v", err)
	}
}

func TestStandbysNeedTwoGigabytes(t *testing.T) {
	list := members(2, 0)
	list[0].MemoryMb, list[1].MemoryMb = 1024, 1024
	if (ClusterMembership{Members: list}).Check() == nil {
		t.Fatal("standbys on 1 GB nodes were accepted")
	}
	list[0].MemoryMb, list[1].MemoryMb = 1740, 1740
	if err := (ClusterMembership{Members: list}).Check(); err != nil {
		t.Fatalf("a 2 GB server reporting 1.7 GB was refused: %v", err)
	}
	single := members(1, 0)
	single[0].MemoryMb = 512
	if err := (NodeConfigure{Ip: single[0].Ip, Members: single}).Check(); err != nil {
		t.Fatalf("a small single node was refused: %v", err)
	}
}

func TestOnlyTheSeedIsSeed(t *testing.T) {
	seed, other := "10.0.0.1", "10.0.0.2"
	cases := map[bool]NodeConfigure{
		true:  {Clustered: true, Ip: seed, Primary: &seed},
		false: {Clustered: true, Ip: other, Primary: &seed},
	}
	for want, request := range cases {
		if request.Seed() != want {
			t.Errorf("%+v: %v", request, !want)
		}
	}
	if !(NodeConfigure{Ip: other}).Seed() || (NodeConfigure{Clustered: true, ReadOnly: true, Ip: seed, Primary: &seed}).Seed() {
		t.Fatal("a single node is its own seed and a replica never is")
	}
}
