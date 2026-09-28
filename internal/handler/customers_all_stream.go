package handler

import (
	"bufio"
	"encoding/json"
	"fmt"

	"dmama_api/internal/model"
)

// customersAllFlushBytes matches meterstat_api's streaming flush threshold.
const customersAllFlushBytes = 64 * 1024

// customerAllSource abstracts one region's customer rows so writeCustomersAllStream can be unit
// tested with a fake row source instead of a live database connection. *repository.CustomerAllRows
// satisfies this interface.
type customerAllSource interface {
	Next() bool
	Customer() (model.DMACustomerAll, error)
	Err() error
	Close()
}

// writeCustomersAllStream writes the streamed body for /api/dma/customers-all:
// {"success":true,"year_month":"...","data":[...],"count":N} with count written last because it
// is unknown until the stream ends.
//
// first is the source for regions[0], already opened by the caller before the HTTP response
// stream started (so a DB error there becomes a normal 500 JSON response instead of a broken
// stream). Sources for any remaining regions are opened lazily, one at a time, via open.
//
// On any error (marshal, write, row scan, or opening the next region) the function returns
// immediately without writing the closing "],\"count\":N}" -- the caller has already flushed
// bytes to the client, so the response is intentionally left as unterminated/invalid JSON for the
// client to detect as a failed request and retry, matching meterstat_api's streaming behaviour.
// Every source this function is handed, including first, is closed before it returns.
func writeCustomersAllStream(w *bufio.Writer, yearMonth string, open func(region int) (customerAllSource, error), regions []int, first customerAllSource) (int, error) {
	if _, err := w.WriteString(`{"success":true,"year_month":"` + yearMonth + `","data":[`); err != nil {
		return 0, err
	}

	count := 0
	pending := 0
	isFirstRow := true

	writeRow := func(customer model.DMACustomerAll) error {
		encoded, err := json.Marshal(customer)
		if err != nil {
			return err
		}
		if !isFirstRow {
			if _, err := w.WriteString(","); err != nil {
				return err
			}
			pending++
		}
		isFirstRow = false
		n, err := w.Write(encoded)
		if err != nil {
			return err
		}
		pending += n
		count++
		if pending >= customersAllFlushBytes {
			if err := w.Flush(); err != nil {
				return err
			}
			pending = 0
		}
		return nil
	}

	readRegion := func(source customerAllSource) error {
		defer source.Close()
		for source.Next() {
			customer, err := source.Customer()
			if err != nil {
				return err
			}
			if err := writeRow(customer); err != nil {
				return err
			}
		}
		return source.Err()
	}

	if err := readRegion(first); err != nil {
		return count, err
	}
	if len(regions) > 1 {
		for _, region := range regions[1:] {
			source, err := open(region)
			if err != nil {
				return count, fmt.Errorf("opening region %d: %w", region, err)
			}
			if err := readRegion(source); err != nil {
				return count, err
			}
		}
	}

	if _, err := w.WriteString(fmt.Sprintf(`],"count":%d}`, count)); err != nil {
		return count, err
	}
	if err := w.Flush(); err != nil {
		return count, err
	}
	return count, nil
}
