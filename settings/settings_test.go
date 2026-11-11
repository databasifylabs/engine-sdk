package settings

import (
	"maps"
	"path/filepath"
	"testing"

	"github.com/databasifylabs/engine-sdk/dtos"
)

func testCatalog() Catalog {
	return Catalog{
		Fields: Fields(
			Integer("max_connections", "Max connections", "Clients at once.", 1, 262143, "100").Restarting().TunedFromServer(),
			Size("shared_buffers", "Shared buffers", "Page cache.", 128, 8589934584, "128MB").Restarting().TunedFromServer(),
			Decimal("random_page_cost", "Random page cost", "Planner cost.", 0, 1e308, "4"),
			Choice("jit", "JIT", "Compile queries.", []string{"on", "off"}, "on"),
			Flags("sql_mode", "SQL mode", "Modes.", []string{"STRICT_TRANS_TABLES", "NO_ZERO_DATE"}, ""),
			Text("timezone", "Time zone", "Zone.", `[A-Za-z_]{1,20}(/[A-Za-z_+-]{1,30}){0,2}`, "UTC"),
		),
		Tune: func(resources dtos.Member) map[string]string {
			return map[string]string{"max_connections": "200", "shared_buffers": "1024MB"}
		},
	}
}

func TestOnlyRealLimitsAreChecked(t *testing.T) {
	catalog := testCatalog()
	valid := map[string]string{"max_connections": "50000", "shared_buffers": "64GB", "random_page_cost": "1.1", "jit": "off", "sql_mode": "STRICT_TRANS_TABLES,NO_ZERO_DATE", "timezone": "Asia/Dhaka"}
	if err := catalog.Check(valid); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{"max_connections": "262144", "shared_buffers": "64kB", "jit": "maybe", "sql_mode": "ANSI", "timezone": "x;y", "unknown_key": "1", "random_page_cost": "fast"} {
		if catalog.Check(map[string]string{key: value}) == nil {
			t.Errorf("%s=%s was accepted", key, value)
		}
	}
}

func TestUserSettingsWinOverTunedValues(t *testing.T) {
	effective := Effective(map[string]string{"max_connections": "200", "shared_buffers": "1024MB"}, map[string]string{"max_connections": "500"})

	if !maps.Equal(effective, map[string]string{"max_connections": "500", "shared_buffers": "1024MB"}) {
		t.Fatalf("effective = %v", effective)
	}
}

func TestTunedValuesAreKeptUntilTuned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuned.json")
	catalog := testCatalog()
	small := dtos.Member{Cpus: 1, MemoryMb: 2048, DiskGb: 20}

	first, err := catalog.Tuned(path, small)
	if err != nil {
		t.Fatal(err)
	}
	catalog.Tune = func(dtos.Member) map[string]string { return map[string]string{"max_connections": "999"} }
	kept, _ := catalog.Tuned(path, small)
	retuned, _ := catalog.Retune(path, small)

	if first["max_connections"] != "200" || kept["max_connections"] != "200" || retuned["max_connections"] != "999" {
		t.Fatalf("first = %v, kept = %v, retuned = %v", first, kept, retuned)
	}
}

func TestFieldsShowTheTunedDefaults(t *testing.T) {
	fields := testCatalog().FieldsWith(map[string]string{"max_connections": "200"})

	if fields[0].Default != "200" || fields[2].Default != "4" {
		t.Fatalf("fields = %+v", fields)
	}
}

func TestBytesAcceptSuffixes(t *testing.T) {
	catalog := Catalog{Fields: Fields(Bytes("innodb_buffer_pool_size", "Buffer pool", "Cache.", 5242880, 9.2e18, "128M"))}

	for _, value := range []string{"4G", "4096M", "134217728", "5m"} {
		if err := catalog.Check(map[string]string{"innodb_buffer_pool_size": value}); err != nil {
			t.Errorf("%s: %v", value, err)
		}
	}
	for _, value := range []string{"1M", "4T", "big"} {
		if catalog.Check(map[string]string{"innodb_buffer_pool_size": value}) == nil {
			t.Errorf("%s was accepted", value)
		}
	}
}
