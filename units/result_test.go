package units

import "testing"

func TestEveryResultButSuccessIsAFailure(t *testing.T) {
	for result, want := range map[string]bool{"": false, "success": false, "exit-code": true, "signal": true, "timeout": true, "resources": true} {
		if Failed(result) != want {
			t.Errorf("Failed(%q) = %v", result, !want)
		}
	}
}
