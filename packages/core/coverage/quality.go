package coverage

import "sort"

// QualityReport is the computed set of coverage quality signals for one spec.
// It answers the question "what have you NOT proven?" rather than just counting
// operations that received any response.
type QualityReport struct {
	// NeverExercised: spec operations never reached by any recorded run.
	NeverExercised []string

	// HappyPathOnly: operations that have been exercised but where only 2xx
	// status codes were ever observed. The declared error paths are untested.
	HappyPathOnly []string

	// SecuredNeverAuthed: operations that require authentication credentials
	// but were never executed with auth present.
	SecuredNeverAuthed []string

	// ErrorsNeverTriggered: operations with declared non-2xx responses that
	// have never been observed in any recorded run.
	ErrorsNeverTriggered []ErrorGap
}

// ErrorGap describes the untested error responses for one operation.
type ErrorGap struct {
	Operation string // "GET /users/{id}"
	Missing   []int  // declared status codes never triggered, e.g. [404, 422]
}

// ComputeQuality derives quality signals from accumulated coverage data
// cross-referenced with spec metadata.
//
// declaredStatusCodes maps "METHOD /path" → all HTTP status codes declared in
// the spec's responses object for that operation.
// securedOps is the set of "METHOD /path" strings for operations that require
// at least one security scheme.
func ComputeQuality(report Report, declaredStatusCodes map[string][]int, securedOps map[string]bool) *QualityReport {
	q := &QualityReport{}

	// 1. Never exercised — straight from the report's Missed list.
	if len(report.Missed) > 0 {
		q.NeverExercised = append(q.NeverExercised, report.Missed...)
		sort.Strings(q.NeverExercised)
	}

	// Build a set of exercised operation keys for fast lookup.
	exercisedSet := make(map[string]bool, len(report.Exercised))
	for _, op := range report.Exercised {
		exercisedSet[op] = true
	}

	// 2. Happy-path only & 3. Error responses never triggered.
	//    Both require knowing which status codes are declared per operation.
	for op, declared := range declaredStatusCodes {
		if !exercisedSet[op] {
			continue // never exercised — already captured above
		}
		seen := statusCodeSet(report.OperationResults[op].StatusCodes)

		// Collect declared non-2xx codes that were never seen.
		var missingErrors []int
		for _, code := range declared {
			if code >= 200 && code < 300 {
				continue // success codes — not error paths
			}
			if !seen[code] {
				missingErrors = append(missingErrors, code)
			}
		}

		// Happy-path only: exercised, but only 2xx ever returned.
		onlyHappy := true
		for code := range seen {
			if code < 200 || code >= 300 {
				onlyHappy = false
				break
			}
		}
		// Also count as happy-path-only if there ARE declared errors but none seen.
		if onlyHappy && len(missingErrors) > 0 {
			q.HappyPathOnly = append(q.HappyPathOnly, op)
		}

		if len(missingErrors) > 0 {
			q.ErrorsNeverTriggered = append(q.ErrorsNeverTriggered, ErrorGap{
				Operation: op,
				Missing:   missingErrors,
			})
		}
	}

	// 4. Secured but never tested with auth.
	for op := range securedOps {
		if !exercisedSet[op] {
			continue // never exercised — already captured
		}
		if !report.OperationResults[op].HadAuth {
			q.SecuredNeverAuthed = append(q.SecuredNeverAuthed, op)
		}
	}

	sort.Strings(q.HappyPathOnly)
	sort.Strings(q.SecuredNeverAuthed)
	sort.Slice(q.ErrorsNeverTriggered, func(i, j int) bool {
		return q.ErrorsNeverTriggered[i].Operation < q.ErrorsNeverTriggered[j].Operation
	})

	return q
}

func statusCodeSet(codes []int) map[int]bool {
	m := make(map[int]bool, len(codes))
	for _, c := range codes {
		m[c] = true
	}
	return m
}
