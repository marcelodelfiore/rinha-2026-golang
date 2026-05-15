package vector

import "math"

func QuantizeFloat32ToUint8(value float32) uint8 {
	if value <= 0 {
		return 0
	}

	if value >= 1 {
		return 255
	}

	return uint8(math.Round(float64(value * 255)))
}

func QuantizeToUint8(src Vector, dst []uint8) {
	for i, value := range src {
		dst[i] = QuantizeFloat32ToUint8(value)
	}
}
