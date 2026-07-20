// Implements: REQ-004.
// Per: ADR-0053, ADR-0061.
// Discipline: C-14.
package pathsegment

import (
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

const opaqueIDPrefix = "id-"

// MaxOpaqueIDBytes bounds the decoded identity before its path expansion.
const MaxOpaqueIDBytes = 1024

// EncodeOpaqueID turns an opaque entity identity into one canonical URL path
// segment. The result contains only lowercase ASCII and never relies on an
// encoded separator surviving an HTTP stack.
func EncodeOpaqueID(entityID string) (string, bool) {
	if entityID == "" || len(entityID) > MaxOpaqueIDBytes || !utf8.ValidString(entityID) || strings.ContainsRune(entityID, '\x00') {
		return "", false
	}
	return opaqueIDPrefix + hex.EncodeToString([]byte(entityID)), true
}

// DecodeOpaqueID validates and decodes one canonical opaque-ID path segment.
// Raw IDs, uppercase hex, percent escapes, and non-canonical aliases fail
// closed; callers must never guess an alternate representation.
func DecodeOpaqueID(segment string) (string, bool) {
	hexValue, ok := strings.CutPrefix(segment, opaqueIDPrefix)
	if !ok || hexValue == "" || len(hexValue) > 2*MaxOpaqueIDBytes || hexValue != strings.ToLower(hexValue) {
		return "", false
	}
	decoded, err := hex.DecodeString(hexValue)
	if err != nil || len(decoded) == 0 || !utf8.Valid(decoded) || strings.IndexByte(string(decoded), 0) >= 0 {
		return "", false
	}
	entityID := string(decoded)
	canonical, ok := EncodeOpaqueID(entityID)
	if !ok || canonical != segment {
		return "", false
	}
	return entityID, true
}
