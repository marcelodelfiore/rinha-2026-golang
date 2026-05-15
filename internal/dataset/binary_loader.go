package dataset

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	binaryMagic   = "R26B"
	binaryVersion = uint32(1)
)

type ReferenceDataset struct {
	Count   int
	Dims    int
	Vectors []float32
	Labels  []uint8
}

func LoadBinary(path string) (*ReferenceDataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open binary references: %w", err)
	}
	defer f.Close()

	header, err := readHeader(f)
	if err != nil {
		return nil, err
	}

	if header.Dims <= 0 {
		return nil, fmt.Errorf("invalid dimensions: %d", header.Dims)
	}

	if header.Count == 0 {
		return nil, errors.New("binary references file has zero references")
	}

	count := int(header.Count)
	dims := int(header.Dims)

	vectors := make([]float32, count*dims)
	labels := make([]uint8, count)

	for i := 0; i < count; i++ {
		offset := i * dims

		for j := 0; j < dims; j++ {
			if err := binary.Read(f, binary.LittleEndian, &vectors[offset+j]); err != nil {
				return nil, fmt.Errorf("read vector[%d][%d]: %w", i, j, err)
			}
		}

		if err := binary.Read(f, binary.LittleEndian, &labels[i]); err != nil {
			return nil, fmt.Errorf("read label[%d]: %w", i, err)
		}
	}

	return &ReferenceDataset{
		Count:   count,
		Dims:    dims,
		Vectors: vectors,
		Labels:  labels,
	}, nil
}

type binaryHeader struct {
	Count uint64
	Dims  uint32
}

func readHeader(r io.Reader) (*binaryHeader, error) {
	magic := make([]byte, 4)

	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}

	if string(magic) != binaryMagic {
		return nil, fmt.Errorf("invalid binary references magic: got %q, want %q", string(magic), binaryMagic)
	}

	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}

	if version != binaryVersion {
		return nil, fmt.Errorf("unsupported binary references version: got %d, want %d", version, binaryVersion)
	}

	var count uint64
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	var dims uint32
	if err := binary.Read(r, binary.LittleEndian, &dims); err != nil {
		return nil, fmt.Errorf("read dimensions: %w", err)
	}

	return &binaryHeader{
		Count: count,
		Dims:  dims,
	}, nil
}
