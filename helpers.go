package main

import (
	"encoding/binary"
	"github.com/go-interpreter/wagon/exec"
	"math"
	"reflect"
	"strings"
)

func getUInt64(proc *exec.Process, addr int32) uint64 {
	data := make([]byte, 8)
	_, _ = proc.ReadAt(data, int64(addr))

	return binary.LittleEndian.Uint64(data)
}

func getUInt32(proc *exec.Process, addr int32) uint32 {
	data := make([]byte, 4)
	_, _ = proc.ReadAt(data, int64(addr))

	return binary.LittleEndian.Uint32(data)
}

func getInt64(proc *exec.Process, addr int32) int64 {
	return int64(getUInt64(proc, addr))
}

func getFloat64(proc *exec.Process, addr int32) float64 {
	return math.Float64frombits(getUInt64(proc, addr))
}

func setUInt64(proc *exec.Process, addr int32, val uint64) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, val)

	_, _ = proc.WriteAt(data, int64(addr))
}

func setInt64(proc *exec.Process, addr int32, val int64) {
	setUInt64(proc, addr, uint64(val))
}

func setUInt32(proc *exec.Process, addr int32, val uint32) {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, val)

	_, _ = proc.WriteAt(data, int64(addr))
}

func setInt32(proc *exec.Process, addr int32, val int32) {
	setUInt32(proc, addr, uint32(val))
}

func setUInt8(proc *exec.Process, addr int32, val uint8) {
	data := []byte{val}
	_, _ = proc.WriteAt(data, int64(addr))
}

func reflectValuesToInterface(vals []reflect.Value) []interface{} {
	islice := make([]interface{}, len(vals))

	for i, v := range vals {
		islice[i] = v.Interface()
	}

	return islice
}

func switchPublicPrivate(name string) string {
	newName := strings.ToLower(name[0:1]) + name[1:]
	if newName == name {
		return strings.ToUpper(name[0:1]) + name[1:]
	}

	return newName
}
