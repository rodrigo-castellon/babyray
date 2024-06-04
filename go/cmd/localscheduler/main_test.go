package main

import (
	"testing"
)

// Unit test for TruncateHighBit
func TestTruncateHighBit(t *testing.T) {
	tests := []struct {
		input    uint64
		expected uint64
	}{
		{0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		{0x8000000000000000, 0x0000000000000000},
		{0x7FFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		{0x0000000000000001, 0x0000000000000001},
	}

	for _, test := range tests {
		result := TruncateHighBit(test.input)
		if result != test.expected {
			t.Errorf("TruncateHighBit(%d) = %d; expected %d", test.input, result, test.expected)
		}
	}
}
