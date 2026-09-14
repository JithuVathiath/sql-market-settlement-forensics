package secftd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func Normalize(inputPath, outputPath, rejectPath string) (NormalizeStats, error) {
	inputs, err := discoverInputs(inputPath)
	if err != nil {
		return NormalizeStats{}, err
	}
	if len(inputs) == 0 {
		return NormalizeStats{}, fmt.Errorf("no .zip, .txt, or .csv inputs found in %s", inputPath)
	}
	if err := ensureParent(outputPath); err != nil {
		return NormalizeStats{}, err
	}
	if err := ensureParent(rejectPath); err != nil {
		return NormalizeStats{}, err
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return NormalizeStats{}, err
	}
	defer out.Close()
	rejects, err := os.Create(rejectPath)
	if err != nil {
		return NormalizeStats{}, err
	}
	defer rejects.Close()

	w := csv.NewWriter(out)
	rw := csv.NewWriter(rejects)
	defer w.Flush()
	defer rw.Flush()
	_ = w.Write([]string{"settlement_date", "cusip", "symbol", "quantity_fails", "description", "price", "source_file", "source_row"})
	_ = rw.Write([]string{"source_file", "source_row", "reason", "raw"})

	var stats NormalizeStats
	for _, source := range inputs {
		stats.Files++
		sourceStats := SourceNormalizeStats{SourceFile: source.name}
		r, err := source.open()
		if err != nil {
			return stats, fmt.Errorf("open %s: %w", source.name, err)
		}
		parser, err := NewParser(source.name, r)
		if err != nil {
			r.Close()
			return stats, fmt.Errorf("parse %s: %w", source.name, err)
		}
		for {
			record, rejected, parseErr := parser.Next()
			if parseErr == io.EOF {
				break
			}
			if parseErr != nil {
				r.Close()
				return stats, parseErr
			}
			if rejected != nil {
				stats.Rejected++
				sourceStats.Rejected++
				_ = rw.Write([]string{rejected.SourceFile, strconv.FormatInt(rejected.SourceRow, 10), rejected.Reason, rejected.Raw})
				continue
			}
			stats.Accepted++
			sourceStats.Accepted++
			_ = w.Write([]string{
				record.SettlementDate.Format("2006-01-02"), record.CUSIP, record.Symbol,
				strconv.FormatInt(record.Quantity, 10), record.Description, record.Price,
				record.SourceFile, strconv.FormatInt(record.SourceRow, 10),
			})
		}
		if err := r.Close(); err != nil {
			return stats, err
		}
		stats.Sources = append(stats.Sources, sourceStats)
	}
	w.Flush()
	rw.Flush()
	if err := w.Error(); err != nil {
		return stats, err
	}
	if err := rw.Error(); err != nil {
		return stats, err
	}
	return stats, nil
}

func WriteNormalizeReceipt(stats NormalizeStats, path string) error {
	if err := ensureParent(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func ensureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
