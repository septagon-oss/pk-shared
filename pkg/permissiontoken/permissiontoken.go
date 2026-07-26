// Package permissiontoken defines the provider-neutral grammar for permission
// declarations shared by PlatformKit-compatible runtimes and clients.
//
// It deliberately owns syntax only. Authorization decisions, role expansion,
// storage, and enforcement remain the responsibility of the consuming runtime.
package permissiontoken

import (
	"fmt"
	"strings"
)

// FullAccess is the canonical full-access grant. It is a distinct grammar from
// resource:action and must remain the bare wildcard.
const FullAccess = "*"

// PermissionExtensionKey is the interoperable OpenAPI operation extension that
// carries a canonical permission declaration.
const PermissionExtensionKey = "x-required-permission"

// TenantScopeParameterExtensionKey is the interoperable OpenAPI operation
// extension that names the path parameter used as the authorization namespace.
const TenantScopeParameterExtensionKey = "x-tenant-scope-parameter"

// Token is the canonical split representation of a permission declaration.
type Token struct {
	Resource string
	Action   string
}

// String returns the canonical resource:action wire form.
func (t Token) String() string {
	return t.Resource + ":" + t.Action
}

// FromParts builds a canonical token from its resource and action.
func FromParts(resource, action string) (Token, error) {
	resource = normalizePart(resource)
	action = normalizePart(action)
	if err := validatePart("resource", resource); err != nil {
		return Token{}, err
	}
	if err := validatePart("action", action); err != nil {
		return Token{}, err
	}
	return Token{Resource: resource, Action: action}, nil
}

// Parse accepts resource:action declarations and the bare full-access grant.
// FullAccess is represented as Token{Resource: "*", Action: "*"}.
func Parse(raw string) (Token, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Token{}, fmt.Errorf("permission token is empty")
	}
	if trimmed == FullAccess {
		return Token{Resource: FullAccess, Action: FullAccess}, nil
	}

	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		return Token{}, fmt.Errorf("invalid permission token %q: expected resource:action", raw)
	}
	return FromParts(parts[0], parts[1])
}

// Normalize returns the canonical resource:action form or the bare full-access
// grant. The wildcard is never expanded to *:*.
func Normalize(raw string) (string, error) {
	if strings.TrimSpace(raw) == FullAccess {
		return FullAccess, nil
	}
	token, err := Parse(raw)
	if err != nil {
		return "", err
	}
	return token.String(), nil
}

// NormalizeAll canonicalizes and deduplicates declarations while preserving
// first-seen order. Empty entries are ignored.
func NormalizeAll(tokens []string) ([]string, error) {
	seen := make(map[string]struct{}, len(tokens))
	normalized := make([]string, 0, len(tokens))
	for _, token := range tokens {
		value := strings.TrimSpace(token)
		if value == "" {
			continue
		}
		canonical, err := Normalize(value)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		normalized = append(normalized, canonical)
	}
	return normalized, nil
}

func normalizePart(part string) string {
	return strings.ToLower(strings.TrimSpace(part))
}

func validatePart(kind, value string) error {
	if value == "" {
		return fmt.Errorf("permission %s is empty", kind)
	}
	if value == FullAccess {
		return nil
	}
	for _, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '.', ch == '_', ch == '-':
		default:
			return fmt.Errorf("permission %s %q contains invalid character %q", kind, value, ch)
		}
	}
	return nil
}
