package main

import (
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/sha3"
)

type uint8array struct {
	data []byte
}

func (u8arr *uint8array) Hash() string {
	hasher := sha3.New224()
	hasher.Sum(u8arr.data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (u8arr *uint8array) String() string {
	return fmt.Sprintf("uint8array(%d)", len(u8arr.data))
}

var importUint8Array = uint8array{}
