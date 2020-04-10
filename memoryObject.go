package main

import "github.com/google/uuid"

type memoryObject struct {
	id    string
	value interface{}
}

func genUnique() string {
	u, err := uuid.NewRandom()
	for err != nil {
		u, err = uuid.NewRandom()
	}

	return u.String()
}

func makeMemoryObject(value interface{}) memoryObject {
	return memoryObject{
		id:    genUnique(),
		value: value,
	}
}
