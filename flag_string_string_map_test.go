package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetFlagStringStringMap(t *testing.T) {
	var sm = make(stringmap)
	sm.Set("CONSUL=yes")
	sm.Set("CELERY=no")

	assert.Equal(t, "yes", sm["CONSUL"])
	assert.Equal(t, "no", sm["CELERY"])
}

func TestIterateFlagStringStringMap(t *testing.T) {
	var sm = make(stringmap)
	sm.Set("CONSUL=yes")
	sm.Set("CELERY=no")
	sm.Set("KITTENS=nomoreplease")
	sm.Set("EQUALS=this=that")

	for k, v := range sm {
		if k == "CONSUL" {
			assert.Equal(t, "yes", v)
		} else if k == "CELERY" {
			assert.Equal(t, "no", v)
		} else if k == "KITTENS" {
			assert.Equal(t, "nomoreplease", v)
		} else if k == "EQUALS" {
			assert.Equal(t, "this=that", v)
		}

	}
}
