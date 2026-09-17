import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// PureScript Char and CodeUnits indices denote UTF-16 units. Native strings
// use UTF-8, with WTF-8 for isolated surrogates (including halves of a slice).
func decodeUnitRune(s string) (rune, int) {
	if len(s) >= 3 && s[0] == 0xed && s[1] >= 0xa0 && s[1] <= 0xbf && s[2]&0xc0 == 0x80 {
		return rune(s[0]&0x0f)<<12 | rune(s[1]&0x3f)<<6 | rune(s[2]&0x3f), 3
	}
	return utf8.DecodeRuneInString(s)
}

func unitString(c uint16) string {
	if c >= 0xd800 && c <= 0xdfff {
		return string([]byte{0xed, 0x80 | byte(c>>6&0x3f), 0x80 | byte(c&0x3f)})
	}
	return string(rune(c))
}

func stringUnits(s string) []uint16 {
	units := make([]uint16, 0, Length(s))
	for i := 0; i < len(s); {
		r, size := decodeUnitRune(s[i:])
		if r > 0xffff {
			hi, lo := utf16.EncodeRune(r)
			units = append(units, uint16(hi), uint16(lo))
		} else {
			units = append(units, uint16(r))
		}
		i += size
	}
	return units
}

func FromCharArray(a []string) string {
	var units []uint16
	for _, c := range a {
		units = append(units, stringUnits(c)...)
	}
	var b strings.Builder
	for i := 0; i < len(units); i++ {
		c := units[i]
		if c >= 0xd800 && c <= 0xdbff && i+1 < len(units) && units[i+1] >= 0xdc00 && units[i+1] <= 0xdfff {
			b.WriteRune(utf16.DecodeRune(rune(c), rune(units[i+1])))
			i++
		} else {
			b.WriteString(unitString(c))
		}
	}
	return b.String()
}

func ToCharArray(str string) []string {
	arr := make([]string, 0, Length(str))
	for i := 0; i < len(str); {
		r, size := decodeUnitRune(str[i:])
		if r > 0xffff {
			hi, lo := utf16.EncodeRune(r)
			arr = append(arr, unitString(uint16(hi)), unitString(uint16(lo)))
		} else {
			arr = append(arr, str[i:i+size])
		}
		i += size
	}
	return arr
}

func Singleton(c string) string {
	return c
}

// Return the byte boundary for a UTF-16 index. split marks the midpoint of a
// supplementary scalar, where slicing must produce an isolated surrogate.
func unitBoundary(s string, index int) (offset int, split bool) {
	if index <= 0 {
		return 0, false
	}
	for i, unit := 0, 0; i < len(s); {
		if unit == index {
			return i, false
		}
		r, size := decodeUnitRune(s[i:])
		if r > 0xffff {
			if unit+1 == index {
				return i, true
			}
			unit++
		}
		unit++
		i += size
	}
	return len(s), false
}

func _CharAt(just func(string) interface{}, nothing interface{}, idx int, str string) interface{} {
	if idx < 0 {
		return nothing
	}
	offset, split := unitBoundary(str, idx)
	if offset == len(str) {
		return nothing
	}
	r, size := decodeUnitRune(str[offset:])
	if r > 0xffff {
		hi, lo := utf16.EncodeRune(r)
		if split {
			return just(unitString(uint16(lo)))
		}
		return just(unitString(uint16(hi)))
	}
	return just(str[offset : offset+size])
}

func _ToChar(just func(string) interface{}, nothing interface{}, str string) interface{} {
	if len(str) > 0 {
		r, size := decodeUnitRune(str)
		if size == len(str) && r <= 0xffff {
			return just(str)
		}
	}
	return nothing
}

func Length(s string) int {
	count := 0
	for i := 0; i < len(s); {
		r, size := decodeUnitRune(s[i:])
		count++
		if r > 0xffff {
			count++
		}
		i += size
	}
	return count
}

func CountPrefix(p func(string) bool, str string) int {
	count := 0
	for i := 0; i < len(str); {
		r, size := decodeUnitRune(str[i:])
		if r > 0xffff {
			hi, lo := utf16.EncodeRune(r)
			if !p(unitString(uint16(hi))) {
				return count
			}
			count++
			if !p(unitString(uint16(lo))) {
				return count
			}
		} else if !p(str[i : i+size]) {
			return count
		}
		count++
		i += size
	}
	return count
}

func findUnits(pattern, input []uint16, start int, backwards bool) int {
	if start > len(input)-len(pattern) {
		start = len(input) - len(pattern)
	}
	for i := start; i >= 0 && i+len(pattern) <= len(input); {
		match := true
		for j, c := range pattern {
			if input[i+j] != c {
				match = false
				break
			}
		}
		if match {
			return i
		}
		if backwards {
			i--
		} else {
			i++
		}
	}
	return -1
}

func _IndexOf(just func(int) interface{}, nothing interface{}, x string, s string) interface{} {
	return _IndexOfStartingAt(just, nothing, x, 0, s)
}

func _IndexOfStartingAt(just func(int) interface{}, nothing interface{}, x string, startIdx int, str string) interface{} {
	input, pattern := stringUnits(str), stringUnits(x)
	if startIdx < 0 || startIdx > len(input) || startIdx+len(pattern) > len(input) {
		return nothing
	}
	idx := findUnits(pattern, input, startIdx, false)
	if idx == -1 {
		return nothing
	}
	return just(idx)
}

func _LastIndexOf(just func(int) interface{}, nothing interface{}, x string, s string) interface{} {
	input, pattern := stringUnits(s), stringUnits(x)
	idx := findUnits(pattern, input, len(input), true)
	if idx == -1 {
		return nothing
	}
	return just(idx)
}

func _LastIndexOfStartingAt(just func(int) interface{}, nothing interface{}, x string, startIdx int, str string) interface{} {
	input, pattern := stringUnits(str), stringUnits(x)
	if startIdx < 0 {
		startIdx = 0
	}
	idx := findUnits(pattern, input, startIdx, true)
	if idx == -1 {
		return nothing
	}
	return just(idx)
}

func sliceUnits(str string, start, end int) string {
	if start >= end {
		return ""
	}
	begin, splitBegin := unitBoundary(str, start)
	finish, splitEnd := unitBoundary(str, end)
	if !splitBegin && !splitEnd {
		return str[begin:finish]
	}
	var b strings.Builder
	if splitBegin {
		r, size := decodeUnitRune(str[begin:])
		_, lo := utf16.EncodeRune(r)
		b.WriteString(unitString(uint16(lo)))
		begin += size
	}
	b.WriteString(str[begin:finish])
	if splitEnd {
		r, _ := decodeUnitRune(str[finish:])
		hi, _ := utf16.EncodeRune(r)
		b.WriteString(unitString(uint16(hi)))
	}
	return b.String()
}

func Take(idx int, str string) string {
	if idx <= 0 {
		return ""
	}
	return sliceUnits(str, 0, idx)
}

func Drop(idx int, str string) string {
	if idx <= 0 {
		return str
	}
	return sliceUnits(str, idx, Length(str))
}

func Slice(start int, end int, str string) string {
	length := Length(str)
	if start < 0 {
		start += length
	}
	if end < 0 {
		end += length
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > length {
		start = length
	}
	if end > length {
		end = length
	}
	return sliceUnits(str, start, end)
}

func SplitAt(idx int, str string) map[string]interface{} {
	return map[string]interface{}{"before": Take(idx, str), "after": Drop(idx, str)}
}
