package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplacePlaceholders(t *testing.T) {
	testString := "We are the {{what_are_we}}, And we are the {{what_are_we_also}}"
	expectedOutput := "We are the music makers, And we are the dreamers of dreams"
	testMap := map[string]string{"what_are_we": "music makers", "what_are_we_also": "dreamers of dreams"}
	output := replacePlaceholders([]byte(testString), testMap)
	assert.Equal(t, string(output), expectedOutput)
}
