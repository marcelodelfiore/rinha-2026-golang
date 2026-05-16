package dataset

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	binaryMagic = "R26B"

	BinaryFormatFloat32 = uint32(1)
	BinaryFormatUint8   = uint32(2)

	binaryVersionFloat32 = uint32(1)
	binaryVersionUint8   = uint32(2)
)

type ReferenceDataset struct {
	Count  int
	Dims   int
	Format uint32

	VectorsF32 []float32
	VectorsU8  []uint8
	Labels     []uint8
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

	switch header.Format {
	case BinaryFormatFloat32:
		return loadFloat32Dataset(f, count, dims, header.Format)

	case BinaryFormatUint8:
		return loadUint8Dataset(f, count, dims, header.Format)

	default:
		return nil, fmt.Errorf("unsupported binary references format: %d", header.Format)
	}
}

func loadFloat32Dataset(r io.Reader, count, dims int, format uint32) (*ReferenceDataset, error) {
	vectors := make([]float32, count*dims)
	labels := make([]uint8, count)

	for i := 0; i < count; i++ {
		offset := i * dims

		for j := 0; j < dims; j++ {
			if err := binary.Read(r, binary.LittleEndian, &vectors[offset+j]); err != nil {
				return nil, fmt.Errorf("read float32 vector[%d][%d]: %w", i, j, err)
			}
		}

		if err := binary.Read(r, binary.LittleEndian, &labels[i]); err != nil {
			return nil, fmt.Errorf("read label[%d]: %w", i, err)
		}
	}

	return &ReferenceDataset{
		Count:      count,
		Dims:       dims,
		Format:     format,
		VectorsF32: vectors,
		Labels:     labels,
	}, nil
}

func loadUint8Dataset(r io.Reader, count, dims int, format uint32) (*ReferenceDataset, error) {
	recordSize := dims + 1

	vectors := make([]uint8, count*dims)
	labels := make([]uint8, count)

	const recordsPerChunk = 8192

	chunkRecords := recordsPerChunk
	if count < chunkRecords {
		chunkRecords = count
	}

	buffer := make([]uint8, chunkRecords*recordSize)

	for start := 0; start < count; start += chunkRecords {
		end := start + chunkRecords
		if end > count {
			end = count
		}

		currentRecords := end - start
		currentBytes := currentRecords * recordSize
		chunk := buffer[:currentBytes]

		if _, err := io.ReadFull(r, chunk); err != nil {
			return nil, fmt.Errorf("read uint8 chunk starting at record %d: %w", start, err)
		}

		for i := 0; i < currentRecords; i++ {
			globalIndex := start + i

			srcOffset := i * recordSize
			dstOffset := globalIndex * dims

			copy(
				vectors[dstOffset:dstOffset+dims],
				chunk[srcOffset:srcOffset+dims],
			)

			labels[globalIndex] = chunk[srcOffset+dims]
		}
	}

	return &ReferenceDataset{
		Count:     count,
		Dims:      dims,
		Format:    format,
		VectorsU8: vectors,
		Labels:    labels,
	}, nil
}

type binaryHeader struct {
	Version uint32
	Count   uint64
	Dims    uint32
	Format  uint32
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

	var count uint64
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	var dims uint32
	if err := binary.Read(r, binary.LittleEndian, &dims); err != nil {
		return nil, fmt.Errorf("read dimensions: %w", err)
	}

	var format uint32
	if err := binary.Read(r, binary.LittleEndian, &format); err != nil {
		return nil, fmt.Errorf("read format: %w", err)
	}

	if err := validateHeader(version, format); err != nil {
		return nil, err
	}

	return &binaryHeader{
		Version: version,
		Count:   count,
		Dims:    dims,
		Format:  format,
	}, nil
}

func validateHeader(version, format uint32) error {
	switch format {
	case BinaryFormatFloat32:
		if version != binaryVersionFloat32 {
			return fmt.Errorf(
				"invalid version for float32 format: got %d, want %d",
				version,
				binaryVersionFloat32,
			)
		}

	case BinaryFormatUint8:
		if version != binaryVersionUint8 {
			return fmt.Errorf(
				"invalid version for uint8 format: got %d, want %d",
				version,
				binaryVersionUint8,
			)
		}

	default:
		return fmt.Errorf("unsupported binary references format: %d", format)
	}

	return nil
}

func (d *ReferenceDataset) IsFloat32() bool {
	return d.Format == BinaryFormatFloat32
}

func (d *ReferenceDataset) IsUint8() bool {
	return d.Format == BinaryFormatUint8
}
