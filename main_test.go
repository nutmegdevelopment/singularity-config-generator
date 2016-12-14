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

func TestWriteMap(t *testing.T) {
	m := map[string]string{"key1": "value1", "key2": "value2"}
	s := `"element": {"key1":"value1","key2":"value2"},`
	s2 := WriteMap("element", m)
	assert.Equal(t, s, s2)
}

func TestWriteMapItems(t *testing.T) {
	m := map[string]string{"key1": "value1", "key2": "value2"}
	s := `"key1":"value1","key2":"value2"`
	s2 := WriteMapItems(m)
	assert.Equal(t, s, s2)
}

func TestWriteSlice(t *testing.T) {
	sl := []string{"value1", "value2"}
	s := `"element": ["value1","value2"],`
	s2 := WriteSlice("element", sl)
	assert.Equal(t, s, s2)
}

func TestWriteSliceItems(t *testing.T) {
	sl := []string{"value1", "value2"}
	s := `"value1","value2"`
	s2 := WriteSliceItems(sl)
	assert.Equal(t, s, s2)
}

func TestReplacePlaceholders(t *testing.T) {
	testString := "We are the {{what_are_we}}, And we are the {{what_are_we_also}}"
	expectedOutput := "We are the music makers, And we are the dreamers of dreams"
	testMap := map[string]string{"what_are_we": "music makers", "what_are_we_also": "dreamers of dreams"}
	output := replacePlaceholders([]byte(testString), testMap)
	assert.Equal(t, string(output), expectedOutput)
}
