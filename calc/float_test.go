package calc

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func patternOf(r *big.Rat, width int) uint64 {
	var p uint64
	for i, b := range FloatBitsOf(Value{R: r}, width) {
		if b {
			p |= 1 << i
		}
	}
	return p
}

// TestFloatAgainstGo checks the encoding against Go's own conversion of an
// exact fraction to float64 and float32, which rounds as IEEE 754 does.
func TestFloatAgainstGo(t *testing.T) {
	rats := []*big.Rat{
		big.NewRat(1, 10), big.NewRat(1, 3), big.NewRat(-5, 2), big.NewRat(2, 3),
		big.NewRat(1, 1<<62), big.NewRat(math.MaxInt64, 3),
	}
	// Edge cases: ties, the smallest subnormals and normals, rounding up
	// into the next power of 2, and too big for the format.
	for _, f := range []float64{
		math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64 * 3, 0x1p-1022, 0x1p-1022 * (1 - 0x1p-52),
		math.MaxFloat64, 1 + 0x1p-53, 1 + 3*0x1p-53, 0.1, -1e-300,
		math.SmallestNonzeroFloat32, math.MaxFloat32, 0x1p-126,
	} {
		rats = append(rats, new(big.Rat).SetFloat64(f))
	}
	// Halfway between two float64s, and beyond the largest.
	rats = append(rats,
		new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).SetFloat64(0x1p-53)),
		new(big.Rat).Mul(new(big.Rat).SetFloat64(math.MaxFloat64), big.NewRat(2, 1)),
		new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 1100)),
	)
	rnd := rand.New(rand.NewSource(1))
	for range 5000 {
		n := new(big.Int).Rand(rnd, new(big.Int).Lsh(big.NewInt(1), uint(1+rnd.Intn(200))))
		d := new(big.Int).Rand(rnd, new(big.Int).Lsh(big.NewInt(1), uint(1+rnd.Intn(200))))
		d.Add(d, big.NewInt(1))
		r := new(big.Rat).SetFrac(n, d)
		if rnd.Intn(2) == 0 {
			r.Neg(r)
		}
		rats = append(rats, r)
	}
	for _, r := range rats {
		f64, _ := r.Float64()
		if got, want := patternOf(r, 64), math.Float64bits(f64); got != want {
			t.Errorf("%s as float64: %016x, want %016x", r, got, want)
		}
		f32, _ := r.Float32()
		if got, want := patternOf(r, 32), uint64(math.Float32bits(f32)); got != want {
			t.Errorf("%s as float32: %08x, want %08x", r, got, want)
		}
	}
}

// TestHalfAndFP8 checks the formats Go has no conversion to against known
// patterns.
func TestHalfAndFP8(t *testing.T) {
	for _, c := range []struct {
		r     *big.Rat
		width int
		want  uint64
	}{
		{big.NewRat(1, 10), 16, 0x2E66},
		{big.NewRat(1, 3), 16, 0x3555},
		{big.NewRat(-5, 2), 16, 0xC100},
		{big.NewRat(65520, 1), 16, 0x7C00}, // rounds up past the largest: infinity
		{big.NewRat(1, 1<<24), 16, 0x0001}, // the smallest subnormal
		{big.NewRat(1, 1<<25), 16, 0x0000}, // half of it ties to even: zero
		{big.NewRat(3, 1<<26), 16, 0x0001}, // three quarters of it rounds up
		{big.NewRat(1, 10), 8, 0x2E},       // FP8 E5M2 is the top byte of half
		{big.NewRat(-3, 2), 8, 0xBE},       // 1.5 = 1.1₂ × 2⁰
		{big.NewRat(1, 1<<16), 8, 0x01},    // the smallest FP8 subnormal
		{big.NewRat(60000, 1), 8, 0x7B},    // below 61440, halfway to 2^16: the largest, 57344
		{big.NewRat(62000, 1), 8, 0x7C},    // beyond it: infinity
		{big.NewRat(0, 1), 64, 0},          // zero
		{new(big.Rat).Neg(big.NewRat(0, 1)), 64, 0},
	} {
		if got := patternOf(c.r, c.width); got != c.want {
			t.Errorf("%s in %d bits: %x, want %x", c.r, c.width, got, c.want)
		}
	}
}

func TestIsFloat(t *testing.T) {
	e := New()
	press(e, "7")
	if e.IsFloat() {
		t.Error("7 is not a fraction")
	}
	press(e, "/2=")
	if !e.IsFloat() {
		t.Error("3.5 is shown as floating point")
	}
	e.SetBase(16)
	if e.IsFloat() {
		t.Error("HEX has no fractions")
	}
}
