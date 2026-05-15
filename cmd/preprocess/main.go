package main

import (
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"
)

const (
	magic = "R26B"

	formatFloat32 = uint32(1)
	formatUint8   = uint32(2)

	versionFloat32 = uint32(1)
	versionUint8   = uint32(2)

	dims = uint32(14)
)

type outputFormat string

const (
	outputFormatFloat32 outputFormat = "float32"
	outputFormatUint8   outputFormat = "uint8"
)

// IMPORTANT:
//
// This struct is intentionally generic.
// Adapt it if your references.json.gz schema uses different fields.
//
// Supported examples:
//
//	{
//	  "vector": [0.1, 0.2, ...],
//	  "fraud": true
//	}
//
//	{
//	  "features": [0.1, 0.2, ...],
//	  "label": "fraud"
//	}
//
//	{
//	  "values": [0.1, 0.2, ...],
//	  "is_fraud": false
//	}
type rawReference struct {
	Vector   []float32 `json:"vector"`
	Features []float32 `json:"features"`
	Values   []float32 `json:"values"`

	Fraud   *bool  `json:"fraud"`
	IsFraud *bool  `json:"is_fraud"`
	Label   string `json:"label"`
	Class   string `json:"class"`
}

func main() {
	referencesPath := flag.String(
		"references",
		"resources/references.json.gz",
		"path to references.json.gz",
	)

	outPath := flag.String(
		"out",
		"resources/references.bin",
		"path to output binary file",
	)

	formatArg := flag.String(
		"format",
		string(outputFormatFloat32),
		"output format: float32 or uint8",
	)

	flag.Parse()

	format, err := parseOutputFormat(*formatArg)
	if err != nil {
		log.Fatal(err)
	}

	if err := run(*referencesPath, *outPath, format); err != nil {
		log.Fatal(err)
	}
}

func parseOutputFormat(value string) (outputFormat, error) {
	normalized := outputFormat(strings.ToLower(strings.TrimSpace(value)))

	switch normalized {
	case outputFormatFloat32, outputFormatUint8:
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid format %q: expected float32 or uint8", value)
	}
}

func run(referencesPath, outPath string, format outputFormat) error {
	log.Printf("opening references: %s", referencesPath)
	log.Printf("output path: %s", outPath)
	log.Printf("output format: %s", format)

	in, err := os.Open(referencesPath)
	if err != nil {
		return fmt.Errorf("open references: %w", err)
	}
	defer in.Close()

	gz, err := gzip.NewReader(in)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gz.Close()

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	if err := writeHeader(out, 0, format); err != nil {
		return fmt.Errorf("write placeholder header: %w", err)
	}

	dec := json.NewDecoder(gz)

	count, err := streamReferences(dec, out, format)
	if err != nil {
		return err
	}

	if err := rewriteHeader(out, count, format); err != nil {
		return fmt.Errorf("rewrite header: %w", err)
	}

	log.Printf("done: wrote %d references to %s", count, outPath)

	return nil
}

// Binary header layout:
//
// magic:   4 bytes  "R26B"
// version: uint32
// count:   uint64
// dims:    uint32
// format:  uint32
//
// Version 1:
//
//	format = 1
//	record = 14 float32 values + 1 uint8 label
//
// Version 2:
//
//	format = 2
//	record = 14 uint8 values + 1 uint8 label
func writeHeader(w io.Writer, count uint64, format outputFormat) error {
	version, formatCode, err := versionAndFormatCode(format)
	if err != nil {
		return err
	}

	if _, err := w.Write([]byte(magic)); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, version); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, count); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, dims); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, formatCode); err != nil {
		return err
	}

	return nil
}

func rewriteHeader(f *os.File, count uint64, format outputFormat) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	return writeHeader(f, count, format)
}

func versionAndFormatCode(format outputFormat) (uint32, uint32, error) {
	switch format {
	case outputFormatFloat32:
		return versionFloat32, formatFloat32, nil
	case outputFormatUint8:
		return versionUint8, formatUint8, nil
	default:
		return 0, 0, fmt.Errorf("unsupported format: %s", format)
	}
}

func streamReferences(dec *json.Decoder, out io.Writer, format outputFormat) (uint64, error) {
	tok, err := dec.Token()
	if err != nil {
		return 0, fmt.Errorf("read first JSON token: %w", err)
	}

	switch delimiter := tok.(type) {
	case json.Delim:
		switch delimiter {
		case '[':
			return streamArray(dec, out, format)

		case '{':
			return streamObjectWithReferencesArray(dec, out, format)

		default:
			return 0, fmt.Errorf("unexpected JSON delimiter: %v", delimiter)
		}

	default:
		return 0, fmt.Errorf("unexpected first JSON token: %v", tok)
	}
}

func streamArray(dec *json.Decoder, out io.Writer, format outputFormat) (uint64, error) {
	count, err := streamArrayBody(dec, out, format)
	if err != nil {
		return count, err
	}

	return count, nil
}

func streamObjectWithReferencesArray(dec *json.Decoder, out io.Writer, format outputFormat) (uint64, error) {
	var total uint64

	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return total, fmt.Errorf("read object key: %w", err)
		}

		key, ok := keyToken.(string)
		if !ok {
			return total, fmt.Errorf("expected object key string, got %T", keyToken)
		}

		if key != "references" && key != "data" && key != "items" {
			if err := skipValue(dec); err != nil {
				return total, fmt.Errorf("skip key %q: %w", key, err)
			}
			continue
		}

		tok, err := dec.Token()
		if err != nil {
			return total, fmt.Errorf("read array token for key %q: %w", key, err)
		}

		delim, ok := tok.(json.Delim)
		if !ok || delim != '[' {
			return total, fmt.Errorf("expected array for key %q", key)
		}

		count, err := streamArrayBody(dec, out, format)
		if err != nil {
			return total, err
		}

		total += count
	}

	if _, err := dec.Token(); err != nil {
		return total, fmt.Errorf("read closing object token: %w", err)
	}

	return total, nil
}

func streamArrayBody(dec *json.Decoder, out io.Writer, format outputFormat) (uint64, error) {
	var count uint64

	for dec.More() {
		var ref rawReference

		if err := dec.Decode(&ref); err != nil {
			return count, fmt.Errorf("decode reference %d: %w", count, err)
		}

		if err := writeReference(out, ref, format); err != nil {
			return count, fmt.Errorf("write reference %d: %w", count, err)
		}

		count++

		if count%100_000 == 0 {
			log.Printf("processed %d references", count)
		}
	}

	if _, err := dec.Token(); err != nil {
		return count, fmt.Errorf("read closing array token: %w", err)
	}

	return count, nil
}

func skipValue(dec *json.Decoder) error {
	var raw json.RawMessage
	return dec.Decode(&raw)
}

func writeReference(out io.Writer, ref rawReference, format outputFormat) error {
	vector, err := extractVector(ref)
	if err != nil {
		return err
	}

	label := extractLabel(ref)

	switch format {
	case outputFormatFloat32:
		return writeReferenceFloat32(out, vector, label)

	case outputFormatUint8:
		return writeReferenceUint8(out, vector, label)

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeReferenceFloat32(out io.Writer, vector []float32, label uint8) error {
	for _, value := range vector {
		if err := binary.Write(out, binary.LittleEndian, value); err != nil {
			return err
		}
	}

	if err := binary.Write(out, binary.LittleEndian, label); err != nil {
		return err
	}

	return nil
}

func writeReferenceUint8(out io.Writer, vector []float32, label uint8) error {
	for _, value := range vector {
		q := quantizeFloat32ToUint8(value)

		if err := binary.Write(out, binary.LittleEndian, q); err != nil {
			return err
		}
	}

	if err := binary.Write(out, binary.LittleEndian, label); err != nil {
		return err
	}

	return nil
}

func quantizeFloat32ToUint8(value float32) uint8 {
	if value <= 0 {
		return 0
	}

	if value >= 1 {
		return 255
	}

	return uint8(math.Round(float64(value * 255)))
}

func extractVector(ref rawReference) ([]float32, error) {
	var vector []float32

	switch {
	case len(ref.Vector) > 0:
		vector = ref.Vector
	case len(ref.Features) > 0:
		vector = ref.Features
	case len(ref.Values) > 0:
		vector = ref.Values
	default:
		return nil, errors.New("reference has no vector/features/values field")
	}

	if len(vector) != int(dims) {
		return nil, fmt.Errorf("invalid vector dimension: got %d, want %d", len(vector), dims)
	}

	return vector, nil
}

func extractLabel(ref rawReference) uint8 {
	if ref.Fraud != nil && *ref.Fraud {
		return 1
	}

	if ref.IsFraud != nil && *ref.IsFraud {
		return 1
	}

	switch strings.ToLower(strings.TrimSpace(ref.Label)) {
	case "fraud", "fraudulent", "1", "true":
		return 1
	}

	switch strings.ToLower(strings.TrimSpace(ref.Class)) {
	case "fraud", "fraudulent", "1", "true":
		return 1
	}

	return 0
}
