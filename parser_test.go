package ecfformat

import (
	"fmt"
	"testing"
)

type confFile struct {
	Name       string
	Birthdate  string
	IdNumber   int
	Employed   bool
	HourlyRate float32

	Hobbies    []string
	WorkPlaces map[string][]string

	Languages map[string]language

	//Class class
}

type language struct {
	Name   string
	Items  []string
	Native string
}

type class struct {
	Name    string
	Items   []string
	Weekend bool
}

func TestKeyValuePairs(t *testing.T) {
	var conf confFile
	p := NewParser()
	err := p.ParseFile(&conf, "testFile.ecff")
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}
	fmt.Printf("%#v\n", conf)
}
