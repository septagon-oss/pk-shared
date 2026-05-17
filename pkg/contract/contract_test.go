package contract

// contract_test.go validates the tiny shared identifier normalization contract.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import "testing"

func TestNormalizeID(t *testing.T) {
	t.Parallel()

	if got := NormalizeID(" auth "); got != "auth" {
		t.Fatalf("NormalizeID() = %q", got)
	}
}
