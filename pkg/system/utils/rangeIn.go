package utils

import "math/rand"

func RangeIn(min, max int) int {
	return min + rand.Intn(max-min)
}