package tdms

import (
	"bytes"
	"math"
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TODO: Tests for all the different data types.

func TestFloat128(t *testing.T) {
	tests := []struct {
		name         string
		value        Float128
		wantBytes    []byte
		wantBigFloat *big.Float
		wantFloat64  float64
	}{
		{
			name:  "NaN",
			value: NewFloat128(math.NaN()),
			wantBytes: []byte{
				0x7F, 0xFF, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			wantBigFloat: nil,
			wantFloat64:  math.NaN(),
		},
		{
			name:  "Positive Infinity",
			value: NewFloat128(math.Inf(1)),
			wantBytes: []byte{
				0x7F, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			wantBigFloat: big.NewFloat(math.Inf(1)),
			wantFloat64:  math.Inf(1),
		},
		{
			name:  "Negative Infinity",
			value: NewFloat128(math.Inf(-1)),
			wantBytes: []byte{
				0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			wantBigFloat: big.NewFloat(math.Inf(-1)),
			wantFloat64:  math.Inf(-1),
		},
		{
			name:  "Positive Zero",
			value: NewFloat128(0),
			wantBytes: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			wantBigFloat: big.NewFloat(0),
			wantFloat64:  0,
		},
		{
			name:  "Negative Zero",
			value: NewFloat128(math.Copysign(0, -1)),
			wantBytes: []byte{
				0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			wantBigFloat: big.NewFloat(math.Copysign(0, -1)),
			wantFloat64:  math.Copysign(0, -1),
		},
		{
			name:  "Pi",
			value: NewFloat128(math.Pi),
			wantBytes: []byte{
				0x40, 0x00, 0x92, 0x1f, 0xb5, 0x44, 0x42, 0xd1,
				0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			wantBigFloat: big.NewFloat(math.Pi),
			wantFloat64:  math.Pi,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBytes := float128ToBytes(tt.value)
			if !bytes.Equal(tt.wantBytes, gotBytes) {
				t.Errorf("expected %v, got %v", tt.wantBytes, gotBytes)
			}

			gotBigFloat := tt.value.AsBigFloat()
			if !cmp.Equal(tt.wantBigFloat, gotBigFloat, cmp.Comparer(compareBigFloats)) {
				t.Errorf("expected %v, got %v", tt.wantBigFloat, gotBigFloat)
			}

			gotFloat64 := tt.value.AsFloat64()
			if !cmp.Equal(tt.wantFloat64, gotFloat64, cmpopts.EquateNaNs()) {
				t.Errorf("expected NaN, got %v", gotFloat64)
			}
		})
	}
}

func TestFloat128NaN(t *testing.T) {
	wantBytes := []byte{
		0x7F, 0xFF, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	got := NewFloat128(math.NaN())

	if !got.IsNaN() {
		t.Errorf("expected NaN, got %v", got)
	}

	gotBytes := float128ToBytes(got)
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Errorf("expected %v, got %v", wantBytes, gotBytes)
	}

	gotBigFloat := got.AsBigFloat()
	if gotBigFloat != nil {
		t.Errorf("expected nil, got %v", gotBigFloat)
	}

	gotFloat64 := got.AsFloat64()
	if !math.IsNaN(gotFloat64) {
		t.Errorf("expected NaN, got %v", gotFloat64)
	}
}

func float128ToBytes(f Float128) []byte {
	b := [16]byte(f)
	return b[:]
}

func compareBigFloats(a, b *big.Float) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Cmp(b) == 0
}
