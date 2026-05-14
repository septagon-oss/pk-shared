// Package contract contains tiny shared vocabulary used across PlatformKit OSS
// repos.
package contract

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
