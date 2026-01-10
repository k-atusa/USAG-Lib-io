// test789c : USAG Lib basic io

package io

import (
	"bytes"
	"encoding/base64"
	"strings"
)

type Encoder struct {
	chars     []rune
	revMap    map[rune]int
	threshold int
	escape    rune
}

func (e *Encoder) Init() {
	e.chars = make([]rune, 0, 32164)
	e.revMap = make(map[rune]int)
	e.threshold = 32164
	e.escape = '.'

	// 1. Korean letters
	for i := 0; i < 11172; i++ {
		e.chars = append(e.chars, rune(0xAC00+i))
	}
	// 2. CJK letters
	for i := 0; i < 20992; i++ {
		e.chars = append(e.chars, rune(0x4E00+i))
	}
	// Reverse Map
	for idx, char := range e.chars {
		e.revMap[char] = idx
	}
}

func (e *Encoder) Encode(data []byte, isBase64 bool) string {
	if isBase64 && len(data) == 0 {
		return ""
	}
	if isBase64 {
		return base64.StdEncoding.EncodeToString(data)
	}
	return e.encodeUnicode(data)
}

func (e *Encoder) Decode(data string) []byte {
	data = strings.ReplaceAll(data, "\r", "")
	data = strings.ReplaceAll(data, "\n", "")
	data = strings.ReplaceAll(data, " ", "")
	if data == "" {
		return []byte{}
	}

	runes := []rune(data)
	if runes[0] < 128 && runes[0] != e.escape {
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return []byte{}
		}
		return decoded
	}
	return e.decodeUnicode(runes)
}

func (e *Encoder) encodeUnicode(data []byte) string {
	var result strings.Builder
	acc := 0
	bits := 0

	for _, b := range data {
		acc = (acc << 8) | int(b)
		bits += 8
		for bits >= 15 {
			bits -= 15
			val := (acc >> bits) & 0x7FFF // Upper 15 bits
			if bits == 0 {
				acc = 0
			} else {
				acc &= (1 << bits) - 1
			}

			if val < e.threshold {
				result.WriteRune(e.chars[val])
			} else {
				offset := val - e.threshold
				result.WriteRune(e.escape)
				result.WriteRune(e.chars[offset])
			}
		}
	}

	// Pad leftover
	val := ((acc << 1) | 1) << (14 - bits)
	if val < e.threshold {
		result.WriteRune(e.chars[val])
	} else {
		offset := val - e.threshold
		result.WriteRune(e.escape)
		result.WriteRune(e.chars[offset])
	}
	return result.String()
}

func (e *Encoder) decodeUnicode(runes []rune) []byte {
	var ba bytes.Buffer
	acc := 0
	bits := 0
	n := len(runes)
	i := 0

	for i < n {
		char := runes[i]
		i++
		val := 0

		if char == e.escape {
			if i >= n {
				return nil // escape error
			}
			nextChar := runes[i]
			i++
			val = e.revMap[nextChar] + e.threshold
		} else {
			val = e.revMap[char]
		}

		acc = (acc << 15) | val
		bits += 15

		for bits >= 8 {
			bits -= 8
			byteVal := byte((acc >> bits) & 0xFF)
			if bits == 0 {
				acc = 0
			} else {
				acc &= (1 << bits) - 1
			}
			ba.WriteByte(byteVal)
		}
	}

	// Cut until last 1
	for bits > 0 && (acc&1) == 0 {
		acc >>= 1
		bits--
	}
	if bits > 0 {
		acc >>= 1
		bits--
	}
	for bits >= 8 {
		bits -= 8
		byteVal := byte((acc >> bits) & 0xFF)
		if bits == 0 {
			acc = 0
		} else {
			acc &= (1 << bits) - 1
		}
		ba.WriteByte(byteVal)
	}
	return ba.Bytes()
}
