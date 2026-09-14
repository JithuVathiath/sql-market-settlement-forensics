package secftd

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var cusipPattern = regexp.MustCompile(`^[A-Za-z0-9*@#]{1,9}$`)

var expectedHeader = []string{
	"SETTLEMENT DATE", "CUSIP", "SYMBOL", "QUANTITY (FAILS)", "DESCRIPTION", "PRICE",
}

type Parser struct {
	reader *csv.Reader
	name   string
	row    int64
}

func NewParser(name string, input io.Reader) (*Parser, error) {
	r := csv.NewReader(input)
	r.Comma = '|'
	r.FieldsPerRecord = -1
	r.ReuseRecord = false
	r.TrimLeadingSpace = false
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	if !equalFoldTrim(header, expectedHeader) {
		return nil, fmt.Errorf("unexpected header: %q", header)
	}
	return &Parser{reader: r, name: name, row: 1}, nil
}

func (p *Parser) Next() (Record, *Reject, error) {
	fields, err := p.reader.Read()
	if errors.Is(err, io.EOF) {
		return Record{}, nil, io.EOF
	}
	p.row++
	if err != nil {
		return Record{}, &Reject{SourceFile: p.name, SourceRow: p.row, Reason: err.Error()}, nil
	}
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	if len(fields) == 1 && fields[0] == "" {
		return Record{}, &Reject{SourceFile: p.name, SourceRow: p.row, Reason: "blank row"}, nil
	}
	if len(fields) != len(expectedHeader) {
		return Record{}, &Reject{SourceFile: p.name, SourceRow: p.row, Reason: fmt.Sprintf("expected 6 fields, got %d", len(fields)), Raw: strings.Join(fields, "|")}, nil
	}
	if equalFoldTrim(fields, expectedHeader) {
		return Record{}, &Reject{SourceFile: p.name, SourceRow: p.row, Reason: "repeated header", Raw: strings.Join(fields, "|")}, nil
	}

	date, err := time.Parse("20060102", fields[0])
	if err != nil {
		return Record{}, reject(p, fields, "invalid settlement date"), nil
	}
	if !cusipPattern.MatchString(fields[1]) {
		return Record{}, reject(p, fields, "invalid CUSIP"), nil
	}
	if fields[2] == "" || len(fields[2]) > 32 || containsControl(fields[2]) {
		return Record{}, reject(p, fields, "invalid symbol"), nil
	}
	quantity, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil || quantity < 0 {
		return Record{}, reject(p, fields, "invalid quantity"), nil
	}
	if fields[4] == "" || len(fields[4]) > 256 || containsControl(fields[4]) {
		return Record{}, reject(p, fields, "invalid description"), nil
	}

	price := fields[5]
	if price == "." {
		price = ""
	} else {
		value, ok := new(big.Rat).SetString(price)
		if !ok || value.Sign() <= 0 || len(price) > 32 {
			return Record{}, reject(p, fields, "invalid price"), nil
		}
	}

	return Record{
		SettlementDate: date,
		CUSIP:          strings.ToUpper(fields[1]), Symbol: strings.ToUpper(fields[2]),
		Quantity: quantity, Description: fields[4], Price: price,
		SourceFile: p.name, SourceRow: p.row,
	}, nil, nil
}

func reject(p *Parser, fields []string, reason string) *Reject {
	return &Reject{SourceFile: p.name, SourceRow: p.row, Reason: reason, Raw: strings.Join(fields, "|")}
}

func equalFoldTrim(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !strings.EqualFold(strings.TrimSpace(left[i]), right[i]) {
			return false
		}
	}
	return true
}

func containsControl(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}
