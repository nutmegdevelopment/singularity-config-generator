package main

import (
	"fmt"
	"log"
	"strings"
)

// Define a stringslice type to hold the key/value pairs passed in via the var
// command line flag.
type stringmap map[string]string

// The delimeter between the key and the value.
var varDelimeter = "="

// Now implement the two methods for the flag.Value interface:

// The first method is String() string
func (s *stringmap) String() string {
	return fmt.Sprintf("%s", *s)
}

// The second method is Set(value string) error
func (s *stringmap) Set(v string) error {
	keyvalue := strings.SplitN(v, varDelimeter, 2)
	if len(keyvalue) == 2 {
		(*s)[keyvalue[0]] = keyvalue[1]
	} else {
		log.Fatalf("There were not two parts to the var: %s - the correct format is: key%svalue", v, varDelimeter)
	}
	return nil
}
