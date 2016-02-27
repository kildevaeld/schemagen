package main

import "time"

// This is cool stuff
type TestStruct struct {
	// Name of the thingy
	Name string `format:"uri"`
	Date *time.Time
}

type TestStruct2 struct {
	Age  int64
	Test TestStruct
}
