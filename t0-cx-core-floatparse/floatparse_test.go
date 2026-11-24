package floatparse

import (
	"errors"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
)

// floatParseCases are edge cases for FloatParseBytes against strconv.ParseFloat.
var floatParseCases = []string{
	// Empty, signs and garbage.
	"", "+", "-", ".", "+.", "-.", "e", "e1", "1e", "1e+", "1e-", "1.e", "1..2", "1.2.3", "--1", "+-1",
	" 1", "1 ", "1x", "x1", "0x", "0x.", "0xp1", "0x1", "0x1p", "0x1p+", "0xg", "1\x00", "\xff", "١",
	"0b101", "0o17", "1f", "1d", "1.5D", "1L", "abc", "true", "null",
	// Integers and plain decimals.
	"0", "-0", "+0", "00", "0.0", "-0.0", ".0", "0.", "1", "-1", "+1", "1.", ".5", "-.5", "007", "1.5",
	"0.1", "0.2", "0.3", "333", "123456", "3.14159265358979323846264338327950288419716939937510",
	// Exponents.
	"1e0", "1E0", "1e+0", "1e-0", "1e1", "1e22", "1e23", "1e-22", "1e-23", "1e38", "1e39", "1e-45",
	"1e-46", "1e308", "1e309", "1e-323", "1e-324", "1e-325", "1e400", "1e-400", "1e10000", "1e-10000",
	"1e99999999999999999999", "1e-99999999999999999999", "0e99999", "0e-99999", "0.000000001e9",
	"100000000000000000000000000e-26",
	// Float64 extremes, subnormals and halfway cases.
	"1.7976931348623157e308", "1.7976931348623158e308", "1.7976931348623159e308",
	"-1.7976931348623157e308", "1.797693134862315708145274237317043567981e308",
	"1.797693134862315808e308", "2.2250738585072014e-308", "2.2250738585072011e-308",
	"2.2250738585072012e-308", "4.9406564584124654e-324", "4.9406564584124653e-324",
	"2.4703282292062327e-324", "2.4703282292062328e-324", "2.470328229206232720882e-324", "5e-324",
	"3e-324", "2e-324", "9007199254740993", "9007199254740992", "9007199254740994", "9007199254740995",
	"9007199254740993.0000000000000000000000001", "18014398509481985", "0.1000000000000000055511151231257827",
	"0.1000000000000000055511151231257828", "0.1000000000000000055511151231257826",
	// Float32 extremes, subnormals and halfway cases.
	"3.4028234663852886e38", "3.4028235e38", "3.4028236e38", "3.40282356779733661637539395458142568448e38",
	"3.40282356779733661637539395458142568447e38", "1.17549435e-38", "1.1754942e-38", "1.4e-45",
	"1.401298464324817e-45", "7.006492321624085e-46", "7.006492321624086e-46", "16777217", "16777216",
	"16777218", "16777219", "33554433", "1.00000005960464477539062499", "1.000000059604644775390625",
	"1.00000005960464477539062501",
	// 19 and more mantissa digits.
	"1234567890123456789", "12345678901234567890", "123456789012345678901234567890",
	"0.12345678901234567890123456789", "18446744073709551615", "18446744073709551616",
	"9999999999999999999", "99999999999999999999", "9999999999999999999e-20",
	"1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
	"0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
	"4.9406564584124654417656879286822137236505980e-324",
	"2.47032822920623272088284396434110686182529901307162382212792841250337753635104375932649918180817996189898282347722858865463328355177969898199387398005390939063150356595155702263922908583924491051844359318028499365361525003193704576782492193656236698636584807570015857692699037063119282795585513329278343384093519780155312465972635795746227664652728272200563740064854999770965994704540208281662262378573934507363390079677619305775067401763246736009689513405355374585166611342237666786041621596804619144672918403005300575308490487653917113865916462395249126236538818796362393732804238910186723484976682350898633885879256283027559956575244555072551893136908362547791869486679949683240497058210285131854513962138377228261454376934125320985913276672363281255e-324",
	"2.47032822920623272088284396434110686182529901307162382212792841250337753635104375932649918180817996189898282347722858865463328355177969898199387398005390939063150356595155702263922908583924491051844359318028499365361525003193704576782492193656236698636584807570015857692699037063119282795585513329278343384093519780155312465972635795746227664652728272200563740064854999770965994704540208281662262378573934507363390079677619305775067401763246736009689513405355374585166611342237666786041621596804619144672918403005300575308490487653917113865916462395249126236538818796362393732804238910186723484976682350898633885879256283027559956575244555072551893136908362547791869486679949683240497058210285131854513962138377228261454376934125320985913276672363281249e-324",
	// Hex floats.
	"0x1p0", "0x1P0", "0X1p0", "0x1p-2", "0x1.8p1", "-0x1.8p1", "0x.8p1", "0x1.p1", "0xAp0", "0xap0",
	"0x1p1023", "0x1p1024", "0x1.fffffffffffffp1023", "0x1.fffffffffffff8p1023", "0x1.fffffffffffff7p1023",
	"0x1p-1022", "0x1p-1074", "0x1p-1075", "0x1.8p-1075", "0x1p-1076", "0x1p127", "0x1p128",
	"0x1.fffffep127", "0x1.ffffffp127", "0x1.fffffefp127", "0x1p-149", "0x1p-150", "0x1.8p-150",
	"0x1.000001p0", "0x1.0000008p0", "0x1.0000018p0", "0x1.00000000000008p0", "0x1.00000000000018p0",
	"0x123456789abcdef0123p0", "0x123456789abcdef0123p-100", "0x0p0", "0x0.0p99999", "0x1p99999",
	"0x1p-99999", "0x1e1", "0x1.5", "0x_1p0", "0x1_0p0", "0x1p1_0",
	// Underscores.
	"1_000", "1_000.5", "1_000.000_5", "1_0e1_0", "_1", "1_", "1__0", "1._5", "1_.5", "1e_1", "1e1_",
	"+_1", "-1_0", "0_1", "1_000e-1_0", "0x_1_p_0", "0x1_p0", "0x_p0", "0__x1p0",
	// Inf and NaN spellings.
	"inf", "Inf", "INF", "+inf", "-inf", "infinity", "Infinity", "INFINITY", "+Infinity", "-infinity",
	"infin", "infinit", "infinityy", "in", "i", "infx", "nan", "NaN", "NAN", "nAn", "+nan", "-nan",
	"nanx", "na", "n", "-in", "+i", "inf ", "infinity1",
}

func floatParseResultOfError(err error) FloatParseResult {
	switch {
	case err == nil:
		return FloatParseResultOK
	case errors.Is(err, strconv.ErrRange):
		return FloatParseResultRange
	case errors.Is(err, strconv.ErrSyntax):
		return FloatParseResultSyntax
	}
	return 255
}

// floatParseCompare reports whether FloatParseBytes(text, bitSize) has the
// bits and the error class of strconv.ParseFloat.
func floatParseCompare(tb testing.TB, text string, bitSize int) {
	want, err := strconv.ParseFloat(text, bitSize)
	got, result := FloatParseBytes([]byte(text), bitSize)
	if math.Float64bits(got) != math.Float64bits(want) || result != floatParseResultOfError(err) {
		tb.Errorf("FloatParseBytes(%q, %d) = %v %#x %d, strconv %v %#x %v",
			text, bitSize, got, math.Float64bits(got), result, want, math.Float64bits(want), err)
	}
}

func TestFloatParseBytesCases(t *testing.T) {
	for _, slow := range []bool{false, true} {
		optimize = !slow
		for _, text := range floatParseCases {
			for _, bitSize := range []int{32, 64} {
				floatParseCompare(t, text, bitSize)
			}
		}
	}
	optimize = true
}

func TestFloatParseBytesNil(t *testing.T) {
	if _, result := FloatParseBytes(nil, 64); result != FloatParseResultSyntax {
		t.Errorf("FloatParseBytes(nil) result %d, want syntax", result)
	}
}

func FuzzFloatParseBytes(f *testing.F) {
	for _, text := range floatParseCases {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		floatParseCompare(t, text, 32)
		floatParseCompare(t, text, 64)
	})
}

// TestFloatParseBytesRandom compares several million inputs: random float64
// and float32 bits formatted with 'e', 'f' and 'g' at varied precision, and
// random digit strings with a dot, an exponent and underscores. Every
// eighth input takes the slow decimal path.
func TestFloatParseBytesRandom(t *testing.T) {
	n := 1 << 21
	if testing.Short() {
		n = 1 << 16
	}
	r := rand.New(rand.NewPCG(412, 92))
	formats := []byte{'e', 'f', 'g', 'x'}
	digits := []byte("0123456789")
	var text []byte
	for i := 0; i < n; i++ {
		optimize = i%8 != 0
		f := math.Float64frombits(r.Uint64())
		if i%2 == 1 {
			f = float64(math.Float32frombits(r.Uint32()))
		}
		format := formats[r.IntN(len(formats))]
		precision := r.IntN(30) - 1
		if format == 'f' {
			precision = r.IntN(20) - 1
		}
		floatParseCompare(t, strconv.FormatFloat(f, format, precision, 64), 32)
		floatParseCompare(t, strconv.FormatFloat(f, format, precision, 64), 64)

		text = text[:0]
		if r.IntN(4) == 0 {
			text = append(text, "+-"[r.IntN(2)])
		}
		for j, length := 0, 1+r.IntN(40); j < length; j++ {
			text = append(text, digits[r.IntN(10)])
			switch r.IntN(40) {
			case 0:
				text = append(text, '.')
			case 1:
				text = append(text, '_')
			}
		}
		if r.IntN(2) == 0 {
			text = append(text, "eE"[r.IntN(2)])
			if r.IntN(2) == 0 {
				text = append(text, "+-"[r.IntN(2)])
			}
			text = strconv.AppendInt(text, int64(r.IntN(700)), 10)
		}
		floatParseCompare(t, string(text), 32)
		floatParseCompare(t, string(text), 64)
		if t.Failed() {
			t.FailNow()
		}
	}
	optimize = true
}

func TestFloatParseBytesAllocs(t *testing.T) {
	texts := [][]byte{
		[]byte("1.5"), []byte("0x1.8p-3"), []byte("1_000.25"), []byte("inf"), []byte("1e400"), []byte("x"),
		[]byte("123456789012345678901234567890"), []byte("1" + strings.Repeat("0", 900) + "1e-900"),
	}
	allocs := testing.AllocsPerRun(10, func() {
		for _, text := range texts {
			FloatParseBytes(text, 32)
			FloatParseBytes(text, 64)
		}
	})
	if allocs != 0 {
		t.Errorf("%.0f allocations per run, want 0", allocs)
	}
}

var floatParseBenchTexts = []string{
	"333", "1.5", "0.1", "-2.5e-3", "3.4028235e38", "1.7976931348623157e308", "4.9406564584124654e-324",
	"123456789012345678901234567890", "0x1.8p-3", "1_000.25", "inf",
}

func BenchmarkFloatParseBytes(b *testing.B) {
	texts := make([][]byte, len(floatParseBenchTexts))
	for i, text := range floatParseBenchTexts {
		texts[i] = []byte(text)
	}
	b.Run("floatparse", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				FloatParseBytes(text, 64)
			}
		}
	})
	b.Run("strconv", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, text := range floatParseBenchTexts {
				strconv.ParseFloat(text, 64)
			}
		}
	})
}
