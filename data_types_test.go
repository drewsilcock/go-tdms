package tdms

import (
	"encoding/binary"
	"math/big"
	"slices"
	"testing"
)

func TestParseQuadZero(t *testing.T) {
	zeroBytes := make([]byte, 16)

	result := parseQuad(zeroBytes, binary.BigEndian)
	if result.GetValue().Cmp(big.NewFloat(0)) != 0 {
		t.Errorf("expected 0, got %v", result)
	}

	result = parseQuad(zeroBytes, binary.LittleEndian)
	if result.GetValue().Cmp(big.NewFloat(0)) != 0 {
		t.Errorf("expected 0, got %v", result)
	}
}

func TestParseQuadOne(t *testing.T) {
	// Sign: 0, Exponent: 16383 (bias), Mantissa: 0
	oneBytes := []byte{
		0x3F, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(oneBytes, binary.BigEndian)
	if result.GetValue().Cmp(big.NewFloat(1)) != 0 {
		t.Errorf("expected 1, got %v", result)
	}

	slices.Reverse(oneBytes)
	result = parseQuad(oneBytes, binary.LittleEndian)
	if result.GetValue().Cmp(big.NewFloat(1)) != 0 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestParseQuadTwo(t *testing.T) {
	// Sign: 0, Exponent: 16384, Mantissa: 0
	twoBytes := []byte{
		0x40, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(twoBytes, binary.BigEndian)
	if result.GetValue().Cmp(big.NewFloat(2)) != 0 {
		t.Errorf("expected 2, got %v", result)
	}

	slices.Reverse(twoBytes)
	result = parseQuad(twoBytes, binary.LittleEndian)
	if result.GetValue().Cmp(big.NewFloat(2)) != 0 {
		t.Errorf("expected 2, got %v", result)
	}
}

func TestParseQuadNegativeOne(t *testing.T) {
	// Sign: 1, Exponent: 16383, Mantissa: 0
	negOneBytes := []byte{
		0xBF, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(negOneBytes, binary.BigEndian)
	if result.GetValue().Cmp(big.NewFloat(-1)) != 0 {
		t.Errorf("expected -1, got %v", result)
	}

	slices.Reverse(negOneBytes)
	result = parseQuad(negOneBytes, binary.LittleEndian)
	if result.GetValue().Cmp(big.NewFloat(-1)) != 0 {
		t.Errorf("expected -1, got %v", result)
	}
}

func TestParseQuadPositiveInfinity(t *testing.T) {
	// Sign: 0, Exponent: all 1s (0x7FFF), Mantissa: 0
	posInfBytes := []byte{
		0x7F, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(posInfBytes, binary.BigEndian)
	if !result.GetValue().IsInf() || result.GetValue().Sign() <= 0 {
		t.Errorf("expected +Inf, got %v", result)
	}

	slices.Reverse(posInfBytes)
	result = parseQuad(posInfBytes, binary.LittleEndian)
	if !result.GetValue().IsInf() || result.GetValue().Sign() <= 0 {
		t.Errorf("expected +Inf, got %v", result)
	}
}

func TestParseQuadNegativeInfinity(t *testing.T) {
	// Sign: 1, Exponent: all 1s (0x7FFF), Mantissa: 0
	negInfBytes := []byte{
		0xFF, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(negInfBytes, binary.BigEndian)
	if !result.GetValue().IsInf() || result.GetValue().Sign() >= 0 {
		t.Errorf("expected -Inf, got %v", result)
	}

	slices.Reverse(negInfBytes)
	result = parseQuad(negInfBytes, binary.LittleEndian)
	if !result.GetValue().IsInf() || result.GetValue().Sign() >= 0 {
		t.Errorf("expected -Inf, got %v", result)
	}
}

func TestParseQuadNaN(t *testing.T) {
	// Sign: 0, Exponent: all 1s (0x7FFF), Mantissa: non-zero
	nanBytes := []byte{
		0x7F, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}

	result := parseQuad(nanBytes, binary.BigEndian)
	if !result.IsNaN() {
		t.Errorf("expected nan, got %v", result)
	}

	slices.Reverse(nanBytes)
	result = parseQuad(nanBytes, binary.LittleEndian)
	if !result.IsNaN() {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestParseQuadHalf(t *testing.T) {
	// 0.5 = 1.0 * 2^-1
	// Sign: 0, Exponent: 16382, Mantissa: 0
	halfBytes := []byte{
		0x3F, 0xFE,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(halfBytes, binary.BigEndian)
	expected := big.NewFloat(0.5)
	if result.GetValue().Cmp(expected) != 0 {
		t.Errorf("expected 0.5, got %v", result)
	}

	slices.Reverse(halfBytes)
	result = parseQuad(halfBytes, binary.LittleEndian)
	if result.GetValue().Cmp(expected) != 0 {
		t.Errorf("expected 0.5, got %v", result)
	}
}

func TestParseQuadQuarter(t *testing.T) {
	// 0.25 = 1.0 * 2^-2
	// Sign: 0, Exponent: 16381, Mantissa: 0
	quarterBytes := []byte{
		0x3F, 0xFD,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(quarterBytes, binary.BigEndian)
	expected := big.NewFloat(0.25)
	if result.GetValue().Cmp(expected) != 0 {
		t.Errorf("expected 0.25, got %v", result)
	}

	slices.Reverse(quarterBytes)
	result = parseQuad(quarterBytes, binary.LittleEndian)
	if result.GetValue().Cmp(expected) != 0 {
		t.Errorf("expected 0.25, got %v", result)
	}
}

func TestParseQuadFour(t *testing.T) {
	// 4 = 1.0 * 2^2
	// Sign: 0, Exponent: 16385, Mantissa: 0
	fourBytes := []byte{
		0x40, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(fourBytes, binary.BigEndian)
	expected := big.NewFloat(4)
	if result.GetValue().Cmp(expected) != 0 {
		t.Errorf("expected 4, got %v", result)
	}

	slices.Reverse(fourBytes)
	result = parseQuad(fourBytes, binary.LittleEndian)
	if result.GetValue().Cmp(expected) != 0 {
		t.Errorf("expected 4, got %v", result)
	}
}

func TestParseQuadThree(t *testing.T) {
	// 3 = 1.1 * 2^1 = (1 + 0.5) * 2
	// Sign: 0, Exponent: 16384, Mantissa: 0x8000000000000000000000000000
	// The mantissa 1 in the fractional part represents 0.5
	threeBytes := []byte{
		0x40, 0x00,
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(threeBytes, binary.BigEndian)
	expected := big.NewFloat(3)
	if result.GetValue().Cmp(expected) != 0 {
		t.Errorf("expected 3, got %v", result)
	}

	slices.Reverse(threeBytes)
	result = parseQuad(threeBytes, binary.LittleEndian)
	if result.GetValue().Cmp(expected) != 0 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestParseQuadNegativeTwo(t *testing.T) {
	// -2
	// Sign: 1, Exponent: 16384, Mantissa: 0
	negTwoBytes := []byte{
		0xC0, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	result := parseQuad(negTwoBytes, binary.BigEndian)
	if result.GetValue().Cmp(big.NewFloat(-2)) != 0 {
		t.Errorf("expected -2, got %v", result)
	}

	slices.Reverse(negTwoBytes)
	result = parseQuad(negTwoBytes, binary.LittleEndian)
	if result.GetValue().Cmp(big.NewFloat(-2)) != 0 {
		t.Errorf("expected -2, got %v", result)
	}
}

func BenchmarkParseQuad(b *testing.B) {
	oneBytes := []byte{
		0x3F, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	for b.Loop() {
		parseQuad(oneBytes, binary.BigEndian)
	}
}

func BenchmarkParseQuadLittleEndian(b *testing.B) {
	oneBytes := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0x3F,
	}

	for b.Loop() {
		parseQuad(oneBytes, binary.LittleEndian)
	}
}
