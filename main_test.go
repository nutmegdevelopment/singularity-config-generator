package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeStringJSONSafe(t *testing.T) {
	s := `{"i":"am","a":"test"}`
	sSafe := `{\"i\":\"am\",\"a\":\"test\"}`
	s = makeStringJSONSafe(s)
	assert.Equal(t, s, sSafe)
}
