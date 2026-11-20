package coverage

// Outcomes is where every operation of a spec stands. It is the data behind the four totals of the
// Contracts page and of every Contracts export, so they all agree:
//
//	Covered + NotCovered = Total          (an operation is covered when a scenario calls it)
//	Passing + Failing    = Covered        (by the operation's most recent result)
//
// Coverage alone says an operation was called, not that it works, so a spec can be fully covered
// and still have failing operations.
type Outcomes struct {
	Total      int
	Covered    int
	NotCovered int
	Passing    int
	Failing    int
	// NoResult counts covered operations whose last result was not recorded (data saved before
	// per-operation results existed). Passing + Failing + NoResult = Covered.
	NoResult int
}

// Outcomes classifies the operations of the report.
func (r Report) Outcomes() Outcomes {
	o := Outcomes{Total: r.Total, Covered: r.Covered}
	if o.NotCovered = r.Total - r.Covered; o.NotCovered < 0 {
		o.NotCovered = 0
	}
	for _, op := range r.Exercised {
		if res, ok := r.OperationResults[op]; ok {
			if res.LastPassed {
				o.Passing++
			} else {
				o.Failing++
			}
		}
	}
	if o.NoResult = o.Covered - o.Passing - o.Failing; o.NoResult < 0 {
		o.NoResult = 0
	}
	return o
}

// Share is count as a whole percentage of total, rounded half up; 0 when total is 0.
func Share(count, total int) int {
	if total <= 0 {
		return 0
	}
	return (count*100 + total/2) / total
}
