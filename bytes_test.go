package ptrie

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBytes_LastSharedIndex(t *testing.T) {

	useCases := []struct {
		description string
		fragment1   string
		fragment2   string
		expectIndex int
	}{
		{
			description: "first match",
			fragment1:   "a",
			fragment2:   "abcdef",
			expectIndex: 0,
		},
		{
			description: "second match",
			fragment1:   "ab",
			fragment2:   "abcdef",
			expectIndex: 1,
		},
		{
			description: "mid match",
			fragment1:   "abc",
			fragment2:   "abcdef",
			expectIndex: 2,
		},
		{
			description: "no match",
			fragment1:   "z",
			fragment2:   "abcdef",
			expectIndex: -1,
		},
		{
			description: "last index match",
			fragment1:   "abcdef",
			fragment2:   "abcdef",
			expectIndex: 5,
		},
		{
			description: "next to last index match",
			fragment1:   "abcdefz",
			fragment2:   "abcdefy",
			expectIndex: 5,
		},
		{
			description: "next to last index match",
			fragment1:   "abrz",
			fragment2:   "abcz",
			expectIndex: 1,
		},
	}

	for _, useCase := range useCases {
		bs1 := []byte(useCase.fragment1)
		bs2 := []byte(useCase.fragment2)
		acutualIndex := Bytes(bs1).LastSharedIndex(bs2)
		assert.Equal(t, useCase.expectIndex, acutualIndex, useCase.description)
	}

}
