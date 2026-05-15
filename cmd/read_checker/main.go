package main

import (
	"flag"
	"log"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/dataset"
)

func main() {
	path := flag.String("path", "resources/references_u8.bin", "path to binary references file")
	flag.Parse()

	ds, err := dataset.LoadBinary(*path)
	if err != nil {
		log.Fatalf("failed to load binary dataset: %v", err)
	}

	log.Printf("loaded dataset")
	log.Printf("count=%d", ds.Count)
	log.Printf("dims=%d", ds.Dims)
	log.Printf("format=%d", ds.Format)
	log.Printf("vectors_f32_len=%d", len(ds.VectorsF32))
	log.Printf("vectors_u8_len=%d", len(ds.VectorsU8))
	log.Printf("labels_len=%d", len(ds.Labels))

	if ds.IsUint8() {
		expectedVectors := ds.Count * ds.Dims

		if len(ds.VectorsU8) != expectedVectors {
			log.Fatalf(
				"invalid uint8 vector length: got %d, want %d",
				len(ds.VectorsU8),
				expectedVectors,
			)
		}

		if len(ds.Labels) != ds.Count {
			log.Fatalf(
				"invalid label length: got %d, want %d",
				len(ds.Labels),
				ds.Count,
			)
		}

		log.Printf("first vector=%v", ds.VectorsU8[:ds.Dims])
		log.Printf("first label=%d", ds.Labels[0])
		log.Printf("last vector=%v", ds.VectorsU8[len(ds.VectorsU8)-ds.Dims:])
		log.Printf("last label=%d", ds.Labels[len(ds.Labels)-1])
	}

	log.Printf("binary dataset looks valid")
}
