package permissiontoken

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"canonical":  {input: "users:read", want: "users:read"},
		"normalized": {input: " Users : READ ", want: "users:read"},
		"wildcard":   {input: " * ", want: FullAccess},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := Normalize(test.input)
			if err != nil {
				t.Fatalf("Normalize(%q): %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("Normalize(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestNormalizeRejectsMalformedTokens(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", "users", "users:read:all", "users:r ead", ":read", "users:"} {
		if _, err := Normalize(input); err == nil {
			t.Errorf("Normalize(%q) unexpectedly succeeded", input)
		}
	}
}

func TestNormalizeAllPreservesFirstSeenOrder(t *testing.T) {
	t.Parallel()

	got, err := NormalizeAll([]string{" Users:READ ", "", "users:read", "roles:update", "*"})
	if err != nil {
		t.Fatalf("NormalizeAll: %v", err)
	}
	want := []string{"users:read", "roles:update", "*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeAll = %#v, want %#v", got, want)
	}
}

func TestOperationExtensionKeysAreStable(t *testing.T) {
	t.Parallel()

	if PermissionExtensionKey != "x-required-permission" {
		t.Fatalf("PermissionExtensionKey = %q", PermissionExtensionKey)
	}
	if TenantScopeParameterExtensionKey != "x-tenant-scope-parameter" {
		t.Fatalf("TenantScopeParameterExtensionKey = %q", TenantScopeParameterExtensionKey)
	}
}
