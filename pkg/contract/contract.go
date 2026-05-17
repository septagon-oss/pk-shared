// Package contract contains tiny shared vocabulary used across PlatformKit OSS
// repos.
package contract

// contract.go owns the smallest cross-repo identifiers that would otherwise
// create dependency cycles between OSS layers.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-14 (every Go file declares its purpose).

import "strings"

// ModuleID is the stable identifier of a PlatformKit module.
type ModuleID string

// Version is a semantic version string. Detailed constraint solving lives in
// the owning runtime package.
type Version string

// NormalizeID trims an identifier without imposing product-specific naming
// policy.
func NormalizeID(id ModuleID) ModuleID {
	return ModuleID(strings.TrimSpace(string(id)))
}
