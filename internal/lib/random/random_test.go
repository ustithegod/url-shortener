package random

import "testing"

func TestNewRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "length = 1",
			length: 1,
		},
		{
			name:   "length = 5",
			length: 5,
		},
		{
			name:   "length = 10",
			length: 10,
		},
		{
			name:   "length = 20",
			length: 20,
		},
		{
			name:   "length = 30",
			length: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewRandomString(tt.length)
			if len(result) != tt.length {
				t.Errorf("expected length %d, got %d", tt.length, len(result))
			}
		})
	}
}
