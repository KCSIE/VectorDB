package pkg

import "math"

func dotProduct(a, b []float32) float32 {
	var sum float32
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

func magnitude(v []float32) float32 {
	var sum float32
	for _, val := range v {
		sum += val * val
	}
	return float32(math.Sqrt(float64(sum)))
}

func DotDistance(a, b []float32) float32 {
	return -dotProduct(a, b)
}

func CosineDistance(a, b []float32) float32 {
	dp := dotProduct(a, b)
	magA := magnitude(a)
	magB := magnitude(b)
	return 1 - dp/(magA*magB)
}

func EuclideanDistance(a, b []float32) float32 {
	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return float32(math.Sqrt(float64(sum)))
}
