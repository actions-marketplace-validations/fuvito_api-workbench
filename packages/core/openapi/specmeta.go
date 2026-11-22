package openapi

import (
	"sort"
	"strconv"
)

// SpecMeta contains the parts of a spec needed for coverage quality computation.
type SpecMeta struct {
	// DeclaredStatusCodes maps "METHOD /path" → all HTTP status codes declared
	// in the spec's responses object for that operation.
	DeclaredStatusCodes map[string][]int
	// SecuredOps is the set of "METHOD /path" strings for operations that
	// declare at least one security requirement.
	SecuredOps map[string]bool
}

// ExtractSpecMeta extracts quality-relevant metadata from a loaded spec.
func ExtractSpecMeta(s *Spec) SpecMeta {
	meta := SpecMeta{
		DeclaredStatusCodes: make(map[string][]int),
		SecuredOps:          make(map[string]bool),
	}

	for path, item := range s.doc.Paths.Map() {
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"} {
			op := opFromItem(item, method)
			if op == nil {
				continue
			}
			key := method + " " + path

			if hasSecurityRequirement(s.doc, op) {
				meta.SecuredOps[key] = true
			}

			if op.Responses != nil {
				codeSet := make(map[int]struct{})
				for statusStr := range op.Responses.Map() {
					if statusStr == "default" {
						continue
					}
					code, err := strconv.Atoi(statusStr)
					if err != nil {
						continue
					}
					codeSet[code] = struct{}{}
				}
				if len(codeSet) > 0 {
					codes := make([]int, 0, len(codeSet))
					for c := range codeSet {
						codes = append(codes, c)
					}
					sort.Ints(codes)
					meta.DeclaredStatusCodes[key] = codes
				}
			}
		}
	}

	return meta
}
