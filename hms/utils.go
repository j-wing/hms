package hms

import (
	"math"
	"strings"
)

const ALPHABET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

func ShortURLEncode(n int) string {
	base := len(ALPHABET)
	num_digits := int(1 + math.Floor((math.Log(float64(n)) / math.Log(float64(base)))))

	chars := make([]byte, num_digits)

	var remainder int
	i := 0
	for n > 0 {
		remainder = n % base
		chars[num_digits-i-1] = ALPHABET[remainder]
		n /= base
		i++
	}

	return string(chars)
}

func ShortURLDecode(s string) int {
	base := len(ALPHABET)

	var result int = 0
	var alphabet_index int
	i := 0
	for _, char := range s {
		alphabet_index = strings.IndexRune(ALPHABET, char)
		if alphabet_index == -1 {
			return -1
		}

		power := float64(len(s) - i - 1)
		result += int(math.Pow(float64(base), power)) * alphabet_index
		i++
	}
	return result
}
