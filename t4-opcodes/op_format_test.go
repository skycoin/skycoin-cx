package opcodes

import (
	"strconv"
	"testing"
)

// TestParseDecimal checks parseDecimalInt and parseDecimalUint against
// strconv.ParseInt and strconv.ParseUint, which the str opcodes used before.
func TestParseDecimal(t *testing.T) {
	texts := []string{
		"", "0", "7", "-7", "+7", "-", "+", "--1", "1_000", "0x10", " 1", "1 ", "12a", "007", "-0",
		"127", "128", "-128", "-129", "255", "256", "32767", "32768", "-32768", "-32769", "65535", "65536",
		"2147483647", "2147483648", "-2147483648", "-2147483649", "4294967295", "4294967296",
		"9223372036854775807", "9223372036854775808", "-9223372036854775808", "-9223372036854775809",
		"18446744073709551615", "18446744073709551616", "99999999999999999999999",
	}
	for _, text := range texts {
		for _, bits := range []uint{8, 16, 32, 64} {
			want, err := strconv.ParseInt(text, 10, int(bits))
			got, ok := parseDecimalInt([]byte(text), bits)
			if ok != (err == nil) || (ok && got != want) {
				t.Errorf("parseDecimalInt(%q, %d) = %d, %v, strconv %d, %v", text, bits, got, ok, want, err)
			}
			wantUint, err := strconv.ParseUint(text, 10, int(bits))
			gotUint, ok := parseDecimalUint([]byte(text), bits)
			if ok != (err == nil) || (ok && gotUint != wantUint) {
				t.Errorf("parseDecimalUint(%q, %d) = %d, %v, strconv %d, %v", text, bits, gotUint, ok, wantUint, err)
			}
		}
	}
}
