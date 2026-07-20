// Validates: REQ-004.
// Per: ADR-0053, ADR-0061, ADR-0075.
// Discipline: C-14.
package pathsegment

import (
	"strings"
	"testing"
)

func TestOpaqueIDRoundTripPreservesExactIdentity(t *testing.T) {
	t.Parallel()

	for _, entityID := range []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"Customer.VIP",
		"tenant/encoded%2Fmember",
		"membro-ç/東京%2Fcliente",
		" identity with spaces ",
	} {
		segment, ok := EncodeOpaqueID(entityID)
		if !ok {
			t.Fatalf("EncodeOpaqueID(%q) rejected a valid opaque identity", entityID)
		}
		if strings.ContainsAny(segment, "/%") || segment != strings.ToLower(segment) {
			t.Fatalf("encoded segment %q is not canonical path-safe lowercase ASCII", segment)
		}
		decoded, ok := DecodeOpaqueID(segment)
		if !ok || decoded != entityID {
			t.Fatalf("DecodeOpaqueID(%q) = %q, %t; want %q, true", segment, decoded, ok, entityID)
		}
	}
}

func TestDecodeOpaqueIDRejectsRawAndNonCanonicalSegments(t *testing.T) {
	t.Parallel()

	for _, segment := range []string{
		"",
		"tenant/encoded%2Fmember",
		"tenant%2Fmember",
		"id-",
		"id-ABCDEF",
		"id-not-hex",
		"id-ff",
		"id-00",
		"id-61%2f62",
	} {
		if decoded, ok := DecodeOpaqueID(segment); ok {
			t.Fatalf("DecodeOpaqueID(%q) = %q, true; want rejection", segment, decoded)
		}
	}
}

func TestOpaqueIDCodecRejectsUnboundedOrNonTextIdentity(t *testing.T) {
	t.Parallel()

	for name, entityID := range map[string]string{
		"empty":        "",
		"oversized":    strings.Repeat("a", MaxOpaqueIDBytes+1),
		"invalid utf8": string([]byte{0xff}),
		"nul byte":     "record\x00suffix",
	} {
		t.Run(name, func(t *testing.T) {
			if segment, ok := EncodeOpaqueID(entityID); ok {
				t.Fatalf("EncodeOpaqueID accepted %s identity as %q", name, segment)
			}
		})
	}
	if decoded, ok := DecodeOpaqueID("id-" + strings.Repeat("61", MaxOpaqueIDBytes+1)); ok {
		t.Fatalf("DecodeOpaqueID accepted oversized identity %q", decoded)
	}
}
