package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"dmama_api/internal/model"
)

// fakeCustomerAllSource is an in-memory customerAllSource used to unit test
// writeCustomersAllStream without a database connection.
type fakeCustomerAllSource struct {
	rows   []model.DMACustomerAll
	pos    int
	err    error
	closed bool
}

func (f *fakeCustomerAllSource) Next() bool {
	if f.pos >= len(f.rows) {
		return false
	}
	f.pos++
	return true
}

func (f *fakeCustomerAllSource) Customer() (model.DMACustomerAll, error) {
	return f.rows[f.pos-1], nil
}

func (f *fakeCustomerAllSource) Err() error { return f.err }
func (f *fakeCustomerAllSource) Close()     { f.closed = true }

func textPtrFor(s string) *string { return &s }

func TestWriteCustomersAllStreamProducesValidJSONWithOrderedKeys(t *testing.T) {
	first := &fakeCustomerAllSource{rows: []model.DMACustomerAll{
		{PwaCode: "5531011", DmaID: textPtrFor("1")},
	}}
	second := &fakeCustomerAllSource{rows: []model.DMACustomerAll{
		{PwaCode: "5531012"},
	}}
	open := func(region int) (customerAllSource, error) { return second, nil }

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	count, err := writeCustomersAllStream(w, "256908", open, []int{1, 2}, first)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
	if !first.closed || !second.closed {
		t.Fatal("expected both sources to be closed")
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid JSON, got error %v; body=%s", err, buf.String())
	}

	// Top-level key order: success, year_month, data, count.
	keyOrder := extractTopLevelKeyOrder(t, buf.Bytes())
	want := []string{"success", "year_month", "data", "count"}
	if len(keyOrder) != len(want) {
		t.Fatalf("expected keys %v, got %v", want, keyOrder)
	}
	for i, k := range want {
		if keyOrder[i] != k {
			t.Fatalf("expected key order %v, got %v", want, keyOrder)
		}
	}

	var countValue int
	if err := json.Unmarshal(decoded["count"], &countValue); err != nil || countValue != 2 {
		t.Fatalf("expected count 2 in output, got %s", decoded["count"])
	}
}

func TestWriteCustomersAllStreamEmptyProducesEmptyDataAndZeroCount(t *testing.T) {
	first := &fakeCustomerAllSource{}
	open := func(region int) (customerAllSource, error) { return nil, errors.New("should not be called") }

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	count, err := writeCustomersAllStream(w, "256908", open, []int{1}, first)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}
	if buf.String() != `{"success":true,"year_month":"256908","data":[],"count":0}` {
		t.Fatalf("unexpected body: %s", buf.String())
	}
}

func TestWriteCustomersAllStreamPerRowFieldOrderMatchesModel(t *testing.T) {
	lat := 13.9849
	lng := 101.6893
	prswtusg := 123.45
	first := &fakeCustomerAllSource{rows: []model.DMACustomerAll{
		{
			DmaID:      textPtrFor("1"),
			DmaName:    textPtrFor("DMA-001"),
			PwaCode:    "5531011",
			IsCustomer: textPtrFor("true"),
			Custstat:   textPtrFor("1"),
			Meterstat:  textPtrFor("1"),
			Usetype:    textPtrFor("22"),
			Custname:   textPtrFor("Customer Name"),
			Latitude:   &lat,
			Longitude:  &lng,
			Custaddr:   textPtrFor("Customer address"),
			Custcode:   textPtrFor("5531011000001"),
			Meterno:    textPtrFor("12345678"),
			Mtrrdroute: textPtrFor("01"),
			Mtrseq:     textPtrFor("0001"),
			Metermake:  textPtrFor("ABC"),
			Metersize:  textPtrFor("15"),
			Prswtusg:   &prswtusg,
		},
	}}
	open := func(region int) (customerAllSource, error) { return nil, errors.New("should not be called") }

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if _, err := writeCustomersAllStream(w, "256908", open, []int{1}, first); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var decoded struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}
	if len(decoded.Data) != 1 {
		t.Fatalf("expected 1 row, got %d", len(decoded.Data))
	}
	rowKeys := extractObjectKeyOrder(t, decoded.Data[0])
	wantPrefix := []string{
		"dma_id", "dma_name", "pwa_code", "is_customer", "custstat", "meterstat", "usetype",
		"custname", "latitude", "longitude", "custaddr", "custcode", "meterno", "mtrrdroute",
		"mtrseq", "metermake", "metersize", "prswtusg", "lstwtusg1",
	}
	for i, want := range wantPrefix {
		if rowKeys[i] != want {
			t.Fatalf("row key %d = %q, want %q (full order: %v)", i, rowKeys[i], want, rowKeys)
		}
	}
	if rowKeys[len(rowKeys)-1] != "lstwtusg12" {
		t.Fatalf("expected last key lstwtusg12, got %q", rowKeys[len(rowKeys)-1])
	}
}

func TestWriteCustomersAllStreamStopsAndReturnsErrorOnSecondRegionFailure(t *testing.T) {
	first := &fakeCustomerAllSource{rows: []model.DMACustomerAll{{PwaCode: "5531011"}}}
	second := &fakeCustomerAllSource{err: errors.New("boom")}
	open := func(region int) (customerAllSource, error) { return second, nil }

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	count, err := writeCustomersAllStream(w, "256908", open, []int{1, 2}, first)
	if err == nil {
		t.Fatal("expected an error from the second region")
	}
	if count != 1 {
		t.Fatalf("expected count 1 (only first region's row), got %d", count)
	}
	if json.Valid(buf.Bytes()) {
		t.Fatalf("expected unterminated/invalid JSON after a mid-stream failure, got valid JSON: %s", buf.String())
	}
	if !first.closed || !second.closed {
		t.Fatal("expected both sources to be closed even on failure")
	}
}

func TestWriteCustomersAllStreamStopsWhenOpeningSecondRegionFails(t *testing.T) {
	first := &fakeCustomerAllSource{rows: []model.DMACustomerAll{{PwaCode: "5531011"}}}
	open := func(region int) (customerAllSource, error) { return nil, errors.New("connection refused") }

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	count, err := writeCustomersAllStream(w, "256908", open, []int{1, 2}, first)
	if err == nil {
		t.Fatal("expected an error opening the second region")
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
	if json.Valid(buf.Bytes()) {
		t.Fatalf("expected unterminated/invalid JSON, got valid JSON: %s", buf.String())
	}
}

// extractTopLevelKeyOrder and extractObjectKeyOrder return a JSON object's keys in file order
// using json.Decoder's token stream, since encoding/json map iteration order is randomized and
// would hide a field-order regression.
func extractTopLevelKeyOrder(t *testing.T, data []byte) []string {
	t.Helper()
	return extractObjectKeyOrder(t, data)
}

func extractObjectKeyOrder(t *testing.T, data []byte) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("reading opening token: %v", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		t.Fatalf("expected object, got %v", tok)
	}
	return readObjectKeys(t, dec)
}

// readObjectKeys reads key/value pairs up to (and including) the matching closing '}',
// returning the keys in order. It assumes the opening '{' has already been consumed.
func readObjectKeys(t *testing.T, dec *json.Decoder) []string {
	t.Helper()
	var keys []string
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			t.Fatalf("reading key: %v", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			t.Fatalf("expected string key, got %v", keyTok)
		}
		keys = append(keys, key)
		skipValue(t, dec)
	}
	if _, err := dec.Token(); err != nil { // closing '}'
		t.Fatalf("reading closing brace: %v", err)
	}
	return keys
}

// skipValue consumes exactly one JSON value (scalar, object, or array), discarding its content.
func skipValue(t *testing.T, dec *json.Decoder) {
	t.Helper()
	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("reading value: %v", err)
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return // scalar (string/number/bool/null) already consumed
	}
	switch delim {
	case '{':
		for dec.More() {
			if _, err := dec.Token(); err != nil { // nested key
				t.Fatalf("reading nested key: %v", err)
			}
			skipValue(t, dec)
		}
		if _, err := dec.Token(); err != nil { // closing '}'
			t.Fatalf("reading nested closing brace: %v", err)
		}
	case '[':
		for dec.More() {
			skipValue(t, dec)
		}
		if _, err := dec.Token(); err != nil { // closing ']'
			t.Fatalf("reading closing bracket: %v", err)
		}
	}
}
