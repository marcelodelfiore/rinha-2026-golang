package dataset

const VectorDimensions = 14

type Dataset struct {
	Vectors []float32
	Labels  []bool
	Count   int
}

func NewDataset(count int) *Dataset {
	return &Dataset{
		Vectors: make([]float32, count*VectorDimensions),
		Labels:  make([]bool, count),
		Count:   count,
	}
}

func (d *Dataset) VectorOffset(index int) int {
	return index * VectorDimensions
}
