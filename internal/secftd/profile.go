package secftd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type RankedSecurity struct {
	CUSIP              string  `json:"cusip"`
	Symbol             string  `json:"symbol"`
	Description        string  `json:"description"`
	Observations       int64   `json:"observations"`
	PricedObservations int64   `json:"priced_observations"`
	QuantityFails      int64   `json:"quantity_fails"`
	DollarProxy        float64 `json:"dollar_proxy"`
}

type DailySummary struct {
	Date          string  `json:"date"`
	Observations  int64   `json:"observations"`
	QuantityFails int64   `json:"quantity_fails"`
	DollarProxy   float64 `json:"dollar_proxy"`
}

type Profile struct {
	Dataset                string           `json:"dataset"`
	Rows                   int64            `json:"rows"`
	FirstDate              string           `json:"first_date"`
	LastDate               string           `json:"last_date"`
	SettlementDates        int              `json:"settlement_dates"`
	UniqueCUSIPs           int              `json:"unique_cusips"`
	UniqueSymbols          int              `json:"unique_symbols"`
	QuantityFails          int64            `json:"quantity_fails"`
	DollarProxy            float64          `json:"dollar_proxy"`
	MissingPriceRows       int64            `json:"missing_price_rows"`
	MissingPriceRate       float64          `json:"missing_price_rate"`
	TopTenQuantityShare    float64          `json:"top_ten_quantity_share"`
	TopByQuantity          []RankedSecurity `json:"top_by_quantity"`
	TopByDollarProxy       []RankedSecurity `json:"top_by_dollar_proxy"`
	Daily                  []DailySummary   `json:"daily"`
	InterpretationBoundary string           `json:"interpretation_boundary"`
}

func BuildProfile(inputPath string) (Profile, error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return Profile{}, err
	}
	defer input.Close()
	r := csv.NewReader(input)
	header, err := r.Read()
	if err != nil {
		return Profile{}, err
	}
	wanted := []string{"settlement_date", "cusip", "symbol", "quantity_fails", "description", "price", "source_file", "source_row"}
	if len(header) != len(wanted) {
		return Profile{}, fmt.Errorf("unexpected canonical header")
	}
	for i := range wanted {
		if header[i] != wanted[i] {
			return Profile{}, fmt.Errorf("unexpected canonical header at column %d", i+1)
		}
	}

	profile := Profile{Dataset: "SEC Fails-to-Deliver", InterpretationBoundary: "Descriptive settlement-stress analysis; not evidence of short selling, naked short selling, manipulation, or misconduct."}
	dates := map[string]struct{}{}
	cusips := map[string]struct{}{}
	symbols := map[string]struct{}{}
	security := map[string]*RankedSecurity{}
	daily := map[string]*DailySummary{}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Profile{}, err
		}
		if len(row) != len(wanted) {
			return Profile{}, fmt.Errorf("canonical row has %d fields", len(row))
		}
		if _, err := time.Parse("2006-01-02", row[0]); err != nil {
			return Profile{}, fmt.Errorf("invalid canonical date %q", row[0])
		}
		quantity, err := strconv.ParseInt(row[3], 10, 64)
		if err != nil {
			return Profile{}, fmt.Errorf("invalid canonical quantity %q", row[3])
		}
		var dollar float64
		if row[5] == "" {
			profile.MissingPriceRows++
		} else {
			price, err := strconv.ParseFloat(row[5], 64)
			if err != nil {
				return Profile{}, fmt.Errorf("invalid canonical price %q", row[5])
			}
			dollar = float64(quantity) * price
		}

		profile.Rows++
		profile.QuantityFails += quantity
		profile.DollarProxy += dollar
		dates[row[0]] = struct{}{}
		cusips[row[1]] = struct{}{}
		symbols[row[2]] = struct{}{}
		if profile.FirstDate == "" || row[0] < profile.FirstDate {
			profile.FirstDate = row[0]
		}
		if row[0] > profile.LastDate {
			profile.LastDate = row[0]
		}
		item := security[row[1]]
		if item == nil {
			item = &RankedSecurity{CUSIP: row[1], Symbol: row[2], Description: row[4]}
			security[row[1]] = item
		}
		item.Observations++
		if row[5] != "" {
			item.PricedObservations++
		}
		item.QuantityFails += quantity
		item.DollarProxy += dollar
		day := daily[row[0]]
		if day == nil {
			day = &DailySummary{Date: row[0]}
			daily[row[0]] = day
		}
		day.Observations++
		day.QuantityFails += quantity
		day.DollarProxy += dollar
	}
	if profile.Rows == 0 {
		return Profile{}, fmt.Errorf("canonical input contains no data rows")
	}
	profile.SettlementDates = len(dates)
	profile.UniqueCUSIPs = len(cusips)
	profile.UniqueSymbols = len(symbols)
	profile.MissingPriceRate = float64(profile.MissingPriceRows) / float64(profile.Rows)
	for _, item := range security {
		profile.TopByQuantity = append(profile.TopByQuantity, *item)
		profile.TopByDollarProxy = append(profile.TopByDollarProxy, *item)
	}
	sort.Slice(profile.TopByQuantity, func(i, j int) bool {
		if profile.TopByQuantity[i].QuantityFails == profile.TopByQuantity[j].QuantityFails {
			return profile.TopByQuantity[i].CUSIP < profile.TopByQuantity[j].CUSIP
		}
		return profile.TopByQuantity[i].QuantityFails > profile.TopByQuantity[j].QuantityFails
	})
	sort.Slice(profile.TopByDollarProxy, func(i, j int) bool {
		if profile.TopByDollarProxy[i].DollarProxy == profile.TopByDollarProxy[j].DollarProxy {
			return profile.TopByDollarProxy[i].CUSIP < profile.TopByDollarProxy[j].CUSIP
		}
		return profile.TopByDollarProxy[i].DollarProxy > profile.TopByDollarProxy[j].DollarProxy
	})
	profile.TopByQuantity = first(profile.TopByQuantity, 10)
	profile.TopByDollarProxy = first(profile.TopByDollarProxy, 10)
	var topQuantity int64
	for _, item := range profile.TopByQuantity {
		topQuantity += item.QuantityFails
	}
	if profile.QuantityFails > 0 {
		profile.TopTenQuantityShare = float64(topQuantity) / float64(profile.QuantityFails)
	}
	for _, day := range daily {
		profile.Daily = append(profile.Daily, *day)
	}
	sort.Slice(profile.Daily, func(i, j int) bool { return profile.Daily[i].Date < profile.Daily[j].Date })
	return profile, nil
}

func first(items []RankedSecurity, n int) []RankedSecurity {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func WriteProfile(profile Profile, jsonPath, markdownPath string) error {
	if err := ensureParent(jsonPath); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	if err := ensureParent(markdownPath); err != nil {
		return err
	}
	return os.WriteFile(markdownPath, []byte(renderProfile(profile)), 0o644)
}

func renderProfile(p Profile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Settlement Forensics: Source Profile\n\n")
	fmt.Fprintf(&b, "> **Interpretation boundary:** %s\n\n", p.InterpretationBoundary)
	fmt.Fprintf(&b, "## Coverage\n\n| Measure | Result |\n|---|---:|\n")
	fmt.Fprintf(&b, "| Rows | %s |\n| Settlement dates | %d |\n| Date range | %s to %s |\n| Unique CUSIPs | %s |\n| Unique symbols | %s |\n", comma(p.Rows), p.SettlementDates, p.FirstDate, p.LastDate, comma(int64(p.UniqueCUSIPs)), comma(int64(p.UniqueSymbols)))
	fmt.Fprintf(&b, "| Aggregate reported balance observations | %s shares |\n| Price × quantity proxy on available-price rows | $%s |\n| Missing-price rows | %s (%.2f%%) |\n| Top-ten quantity concentration | %.2f%% |\n\n", comma(p.QuantityFails), money(p.DollarProxy), comma(p.MissingPriceRows), 100*p.MissingPriceRate, 100*p.TopTenQuantityShare)
	fmt.Fprintf(&b, "## Largest Aggregate Quantity Observations\n\n| Rank | Symbol | CUSIP | Observations | Aggregate quantity | Dollar proxy |\n|---:|---|---|---:|---:|---:|\n")
	for i, item := range p.TopByQuantity {
		fmt.Fprintf(&b, "| %d | %s | `%s` | %s | %s | %s |\n", i+1, item.Symbol, item.CUSIP, comma(item.Observations), comma(item.QuantityFails), rankedMoney(item))
	}
	b.WriteString("\n## Reading These Results\n\n")
	b.WriteString("The quantity field is a daily aggregate net balance. Summing it across dates measures aggregate **reported balance observations**, not unique failed trades. The dollar figure is only a descriptive proxy (`quantity × SEC reference price`) and is not a loss, liability, or transaction value. Review flags generated by the SQL layer are leads for investigation, not allegations.\n")
	return b.String()
}

func rankedMoney(item RankedSecurity) string {
	if item.PricedObservations == 0 {
		return "Unavailable"
	}
	return "$" + money(item.DollarProxy)
}

func comma(value int64) string {
	s := strconv.FormatInt(value, 10)
	start := 0
	if strings.HasPrefix(s, "-") {
		start = 1
	}
	for i := len(s) - 3; i > start; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func money(value float64) string {
	s := fmt.Sprintf("%.2f", value)
	parts := strings.SplitN(s, ".", 2)
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	return comma(whole) + "." + parts[1]
}
