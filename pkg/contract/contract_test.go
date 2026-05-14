package contract

import "testing"

func TestNormalizeID(t *testing.T) {
	t.Parallel()

	if got := NormalizeID(" auth "); got != "auth" {
		t.Fatalf("NormalizeID() = %q", got)
	}
}
