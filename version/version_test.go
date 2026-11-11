package version

import "testing"

func TestNewerThanComparesEachPartAsANumber(t *testing.T) {
	cases := map[[2]string]bool{
		{"9.7.2", "8.4.13"}: true, {"8.4.13", "9.7.2-x"}: false, {"8.4.13", "8.4.13"}: false, {"8.4.20", "8.4.13"}: true,
		{"10.11", "9.1"}: true, {"9.1", "10.11"}: false, {"11.6", "10.11"}: true, {"18", "9"}: true, {"9", "10"}: false,
	}
	for pair, want := range cases {
		if got := NewerThan(pair[0], pair[1]); got != want {
			t.Errorf("%v: %v", pair, got)
		}
	}
}
