package conformance_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
)

// Pins CS-GAP-012: a probe passes on any 4xx, whatever the reason for the rejection. Change this
// test and the catalog together when probes learn to tell the reasons apart.
func TestNegativeProbe_AnyClientErrorPasses_KnownGap(t *testing.T) {
	for _, status := range []int{401, 404, 429} {
		p := conformance.NegativeProbeResult{Kind: "drop_required_field", StatusCode: status}
		if !p.Passed() {
			t.Errorf("status %d no longer passes a missing-field probe: update CS-GAP-012 and the CS-PROBE rows", status)
		}
	}
	accepted := conformance.NegativeProbeResult{Kind: "drop_required_field", StatusCode: 200}
	if accepted.Passed() {
		t.Error("a server that accepts the bad request must fail the probe")
	}
}
