package secftd

import (
	"io"
	"strings"
	"testing"
)

func TestParserValidAndMissingPrice(t *testing.T) {
	input := strings.Join([]string{
		strings.Join(expectedHeader, "|"),
		"20260701|037833100|AAPL|1200|APPLE INC|201.25",
		"20260702|594918104|MSFT|3400|MICROSOFT CORP|.",
	}, "\n")
	parser, err := NewParser("fixture.txt", strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	first, rejected, err := parser.Next()
	if err != nil || rejected != nil {
		t.Fatalf("unexpected parse result: reject=%v err=%v", rejected, err)
	}
	if first.CUSIP != "037833100" || first.Symbol != "AAPL" || first.Price != "201.25" || first.Quantity != 1200 {
		t.Fatalf("unexpected record: %#v", first)
	}
	second, rejected, err := parser.Next()
	if err != nil || rejected != nil {
		t.Fatalf("unexpected parse result: reject=%v err=%v", rejected, err)
	}
	if second.PriceAvailable() {
		t.Fatalf("dot price must normalize to missing: %#v", second)
	}
	if _, _, err := parser.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestParserRejectsMalformedRows(t *testing.T) {
	cases := []struct{ name, row, reason string }{
		{"date", "20261340|037833100|AAPL|1|APPLE INC|1.00", "invalid settlement date"},
		{"cusip", "20260701|bad cusip|AAPL|1|APPLE INC|1.00", "invalid CUSIP"},
		{"symbol", "20260701|037833100||1|APPLE INC|1.00", "invalid symbol"},
		{"quantity", "20260701|037833100|AAPL|-1|APPLE INC|1.00", "invalid quantity"},
		{"description", "20260701|037833100|AAPL|1||1.00", "invalid description"},
		{"price", "20260701|037833100|AAPL|1|APPLE INC|zero", "invalid price"},
		{"fields", "20260701|037833100|AAPL", "expected 6 fields"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := NewParser("bad.txt", strings.NewReader(strings.Join(expectedHeader, "|")+"\n"+tc.row))
			if err != nil {
				t.Fatal(err)
			}
			_, rejected, err := parser.Next()
			if err != nil || rejected == nil || !strings.Contains(rejected.Reason, tc.reason) {
				t.Fatalf("wanted %q rejection, got %#v, %v", tc.reason, rejected, err)
			}
		})
	}
}

func TestParserRejectsUnexpectedHeader(t *testing.T) {
	if _, err := NewParser("bad.txt", strings.NewReader("date|cusip\n")); err == nil {
		t.Fatal("expected header error")
	}
}
