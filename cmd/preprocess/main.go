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
	"os"
)

const (
	magic   = "R26B"
	version = uint32(1)
	dims    = uint32(14)
)

// IMPORTANT:
//
// This struct is intentionally generic.
// You may need to adapt it to the real references.json.gz schema.
//
// Supported examples:
//
// 1)
// {
//   "vector": [0.1, 0.2, ...],
//   "fraud": true
// }
//
// 2)
// {
//   "features": [0.1, 0.2, ...],
//   "label": "fraud"
// }
//
// 3)
// {
//   "values": [0.1, 0.2, ...],
//   "is_fraud": false
// }
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
	referencesPath := flag.String("references", "resources/references.json.gz", "path to references.json.gz")
	outPath := flag.String("out", "resources/references.bin", "path to output binary file")
	flag.Parse()

	if err := run(*referencesPath, *outPath); err != nil {
		log.Fatal(err)
	}
}

func run(referencesPath, outPath string) error {
	log.Printf("opening references: %s", referencesPath)

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

	if err := writeHeader(out, 0); err != nil {
		return fmt.Errorf("write placeholder header: %w", err)
	}

	dec := json.NewDecoder(gz)

	count, err := streamReferences(dec, out)
	if err != nil {
		return err
	}

	if err := rewriteHeader(out, count); err != nil {
		return fmt.Errorf("rewrite header: %w", err)
	}

	log.Printf("done: wrote %d references to %s", count, outPath)
	return nil
}

func writeHeader(w io.Writer, count uint64) error {
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

	return nil
}

func rewriteHeader(f *os.File, count uint64) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	return writeHeader(f, count)
}

func streamReferences(dec *json.Decoder, out io.Writer) (uint64, error) {
	tok, err := dec.Token()
	if err != nil {
		return 0, fmt.Errorf("read first JSON token: %w", err)
	}

	switch delimiter := tok.(type) {
	case json.Delim:
		switch delimiter {
		case '[':
			return streamArray(dec, out)

		case '{':
			return streamObjectWithReferencesArray(dec, out)

		default:
			return 0, fmt.Errorf("unexpected JSON delimiter: %v", delimiter)
		}

	default:
		return 0, fmt.Errorf("unexpected first JSON token: %v", tok)
	}
}

func streamArray(dec *json.Decoder, out io.Writer) (uint64, error) {
	var count uint64

	for dec.More() {
		var ref rawReference

		if err := dec.Decode(&ref); err != nil {
			return count, fmt.Errorf("decode reference %d: %w", count, err)
		}

		if err := writeReference(out, ref); err != nil {
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

func streamObjectWithReferencesArray(dec *json.Decoder, out io.Writer) (uint64, error) {
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

		count, err := streamArrayBody(dec, out)
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

func streamArrayBody(dec *json.Decoder, out io.Writer) (uint64, error) {
	var count uint64

	for dec.More() {
		var ref rawReference

		if err := dec.Decode(&ref); err != nil {
			return count, fmt.Errorf("decode reference %d: %w", count, err)
		}

		if err := writeReference(out, ref); err != nil {
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

func writeReference(out io.Writer, ref rawReference) error {
	vector, err := extractVector(ref)
	if err != nil {
		return err
	}

	label := extractLabel(ref)

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

	switch ref.Label {
	case "fraud", "fraudulent", "1", "true":
		return 1
	}

	switch ref.Class {
	case "fraud", "fraudulent", "1", "true":
		return 1
	}

	return 0
}
