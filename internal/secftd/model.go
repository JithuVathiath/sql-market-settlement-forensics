package secftd

import "time"

// Record is one SEC-reported aggregate net fails-to-deliver balance for a
// security and settlement date. It is not a record of a new individual fail.
type Record struct {
	SettlementDate time.Time
	CUSIP          string
	Symbol         string
	Quantity       int64
	Description    string
	Price          string
	SourceFile     string
	SourceRow      int64
}

func (r Record) PriceAvailable() bool { return r.Price != "" }

type Reject struct {
	SourceFile string
	SourceRow  int64
	Reason     string
	Raw        string
}

type NormalizeStats struct {
	Files    int64                  `json:"files"`
	Accepted int64                  `json:"accepted"`
	Rejected int64                  `json:"rejected"`
	Sources  []SourceNormalizeStats `json:"sources"`
}

type SourceNormalizeStats struct {
	SourceFile string `json:"source_file"`
	Accepted   int64  `json:"accepted"`
	Rejected   int64  `json:"rejected"`
}
