package main

import (
	"encoding/binary"
	"fmt"
	"github.com/go-interpreter/wagon/exec"
	"io"
	"math"
	"os"
	"reflect"
	"strings"
	"time"
)

const nanHead = 0x7FF80000

var global = map[string]interface{}{
	"fs":         fsImport,
	"process":    importProcess,
	"Uint8Array": importUint8Array,
}
var scope = map[string]interface{}{}

var storedValues = map[int]interface{}{
	0: math.NaN(),
	1: 0,
	2: nil,
	3: true,
	4: false,
	5: global,
	6: scope,
}
var storedIds = map[interface{}]int{}
var idpool = []int{}
var goRefCounts = map[int]int{}

func StoreAsMemoryObject(proc *exec.Process, addr int64, val interface{}) {
	data := make([]byte, 8)
	memObj := makeMemoryObject(val)
	hashValue := memObj.id
	id, ok := storedIds[hashValue]
	if !ok {
		if len(idpool) > 0 {
			id = idpool[len(idpool)-1]
			idpool = idpool[:len(idpool)-1]
		} else {
			id = len(storedValues)
		}
		storedValues[id] = memObj
		goRefCounts[id] = 0
		storedIds[hashValue] = id
	}

	goRefCounts[id]++
	typeFlag := 1 // Object

	binary.LittleEndian.PutUint32(data[4:], uint32(nanHead|typeFlag))
	binary.LittleEndian.PutUint32(data[:4], uint32(id))

	//fmt.Printf("stored as memobj %v in %d\n", val, id)
	_, _ = proc.WriteAt(data, addr)
}

func StoreObject(proc *exec.Process, addr int64, val interface{}) {
	if strings.Contains(reflect.TypeOf(val).String(), "map") {
		StoreAsMemoryObject(proc, addr, val)
		return
	}

	if v, ok := val.(*uint8array); ok {
		StoreAsMemoryObject(proc, addr, v)
		return
	}
	if v, ok := val.(uint8array); ok {
		StoreAsMemoryObject(proc, addr, v)
		return
	}

	data := make([]byte, 8)

	id, ok := storedIds[val]
	if !ok {
		if len(idpool) > 0 {
			id = idpool[len(idpool)-1]
			idpool = idpool[:len(idpool)-1]
		} else {
			id = len(storedValues)
		}
		storedValues[id] = val
		goRefCounts[id] = 0
		storedIds[val] = id
	}

	goRefCounts[id]++
	typeFlag := 1

	switch reflect.TypeOf(val).Kind().String() {
	case "string":
		typeFlag = 2
	case "func":
		typeFlag = 4
	}

	binary.LittleEndian.PutUint32(data[4:], uint32(nanHead|typeFlag))
	binary.LittleEndian.PutUint32(data[:4], uint32(id))

	//fmt.Printf("stored data %v in %d\n", val, id)
	_, _ = proc.WriteAt(data, addr)
}

func StoreValue(proc *exec.Process, addr int64, v interface{}) {
	tmp := make([]byte, 8)

	if v == nil {
		binary.LittleEndian.PutUint32(tmp[4:], nanHead)
		binary.LittleEndian.PutUint32(tmp[:4], 2)
		_, _ = proc.WriteAt(tmp, addr)
		return
	}

	switch vi := v.(type) {
	case int:
		StoreValue(proc, addr, float64(vi))
	case int32:
		StoreValue(proc, addr, float64(vi))
	case int64:
		StoreValue(proc, addr, float64(vi))
	case uint:
		StoreValue(proc, addr, float64(vi))
	case uint32:
		StoreValue(proc, addr, float64(vi))
	case uint64:
		StoreValue(proc, addr, float64(vi))
	case float32:
		StoreValue(proc, addr, float64(vi))
	case float64:
		if math.IsNaN(vi) {
			binary.LittleEndian.PutUint32(tmp[4:], nanHead)
			binary.LittleEndian.PutUint32(tmp[:4], 0)
			_, _ = proc.WriteAt(tmp, addr)
			return
		}
		if vi == 0 {
			binary.LittleEndian.PutUint32(tmp[4:], nanHead)
			binary.LittleEndian.PutUint32(tmp[:4], 1)
			_, _ = proc.WriteAt(tmp, addr)
			return
		}

		viu := math.Float64bits(vi)
		binary.LittleEndian.PutUint64(tmp, viu)
		_, _ = proc.WriteAt(tmp, addr)
	case bool:
		binary.LittleEndian.PutUint32(tmp[4:], nanHead)
		if vi {
			binary.LittleEndian.PutUint32(tmp[:4], 3)
		} else {
			binary.LittleEndian.PutUint32(tmp[:4], 4)
		}
		_, _ = proc.WriteAt(tmp, addr)
	default:
		StoreObject(proc, addr, vi)
	}
}

func LoadValue(proc *exec.Process, p int32) (interface{}, int) {
	f := getFloat64(proc, p)

	if f == 0 {
		return nil, -1
	}

	if !math.IsNaN(f) {
		return f, -1
	}

	id := int(getUInt32(proc, p))

	v := storedValues[id]

	if memObj, ok := v.(memoryObject); ok {
		v = memObj.value
	}

	return v, id
}

func LoadString(proc *exec.Process, p int32) string {
	saddr := getUInt64(proc, p)
	l := getUInt64(proc, p+8)

	data := make([]byte, l)
	_, _ = proc.ReadAt(data, int64(saddr))

	return string(data)
}

func LoadSliceOfValues(proc *exec.Process, p int32) []interface{} {
	arrayPtr := getUInt64(proc, p)
	arrayLen := int(getUInt64(proc, p+8))
	values := make([]interface{}, arrayLen)

	for i := 0; i < arrayLen; i++ {
		values[i], _ = LoadValue(proc, int32(arrayPtr+uint64(i*8)))
	}

	return values
}

func LoadSlice(proc *exec.Process, p int32) []byte {
	arrayPtr := getUInt64(proc, p)
	arrayLen := getUInt64(proc, p+8)

	data := make([]byte, arrayLen)
	_, _ = proc.ReadAt(data, int64(arrayPtr))

	return data
}

type GoHostFunc func(proc *exec.Process, p int32)

func debug(proc *exec.Process, p int32) {
	s := LoadString(proc, p)
	fmt.Printf("DEBUG(%d): %s\n", p, s)
}

func resetMemoryDataView(proc *exec.Process, p int32) {
	fmt.Printf("ResetMemoryDataView(%d)\n", p)
}

func wasmExit(proc *exec.Process, p int32) {
	fmt.Printf("WasmExit(%d)\n", p)
	// runtime.wasmExit
}

func wasmWrite(proc *exec.Process, sp int32) {
	//fmt.Printf("WasmWrite(%d)\n", sp)

	data := make([]byte, 8)

	_, _ = proc.ReadAt(data, int64(sp+8))
	fd := binary.LittleEndian.Uint64(data)

	_, _ = proc.ReadAt(data, int64(sp+16))
	p := binary.LittleEndian.Uint64(data)

	_, _ = proc.ReadAt(data[:4], int64(sp+24))
	n := binary.LittleEndian.Uint32(data)

	var w io.Writer

	switch fd {
	case 0:
		w = os.Stdout
	case 1:
		w = os.Stderr
	case 2:
		w = os.Stdin
	default:
		fmt.Printf("No such FD: %d\n", fd)
		return
	}
	data = make([]byte, n)
	_, _ = proc.ReadAt(data, int64(p))

	_, _ = fmt.Fprint(w, string(data))
}

func nanotime1(proc *exec.Process, p int32) {
	//fmt.Printf("nanotime1(%d)\n", p)
	data := make([]byte, 8)
	v := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(data, uint64(v))

	_, _ = proc.WriteAt(data, int64(p)+8)
}

func walltime1(proc *exec.Process, p int32) {
	//fmt.Printf("walltime1(%d)\n", p)
	data := make([]byte, 8)
	msec := time.Now().UnixNano() / 1e3
	binary.LittleEndian.PutUint64(data, uint64(msec)/1000)
	_, _ = proc.WriteAt(data, int64(p)+8)
	binary.LittleEndian.PutUint32(data, (uint32(msec%1000))*1000000)
	_, _ = proc.WriteAt(data[:4], int64(p)+16)
}

func scheduleTimeoutEvent(proc *exec.Process, p int32) {
	fmt.Printf("scheduleTimeoutEvent(%d)\n", p)
}

func clearTimeoutEvent(proc *exec.Process, p int32) {
	fmt.Printf("clearTimeoutEvent(%d)\n", p)
}

func getRandomData(proc *exec.Process, p int32) {
	//fmt.Printf("getRandomData(%d)\n", p)
}

func finalizeRef(proc *exec.Process, p int32) {
	//fmt.Printf("finalizeRef(%d)\n", p)
	data := make([]byte, 4)
	_, _ = proc.ReadAt(data, int64(p)+8)

	id := int(binary.LittleEndian.Uint32(data))
	if _, ok := goRefCounts[id]; ok {
		goRefCounts[id]--
		if goRefCounts[id] == 0 {
			v := storedValues[id]
			storedValues[id] = nil
			delete(storedIds, v)
			idpool = append(idpool, id)
		}
	}
}

func stringVal(proc *exec.Process, p int32) {
	//fmt.Printf("stringVal(%d)\n", p)
	StoreValue(proc, int64(p)+24, LoadString(proc, p+8))
}

func valueGet(proc *exec.Process, p int32) {
	// fmt.Printf("valueGet(%08x)\n", p)
	obj, _ := LoadValue(proc, p+8)
	key := LoadString(proc, p+16)

	//fmt.Printf("Get %s from %d\n", key, id)

	if obj != nil {
		objVal := reflect.ValueOf(obj)

		if objVal.Type().String() == "map[string]interface {}" {
			m := obj.(map[string]interface{})
			v, ok := m[key]
			if !ok {
				StoreValue(proc, int64(p)+32, nil)
				return
			}
			StoreValue(proc, int64(p)+32, v)
			return
		}

		fieldVal := objVal.FieldByName(key).Interface()
		StoreValue(proc, int64(p)+32, fieldVal)
	} else {
		StoreValue(proc, int64(p)+32, nil)
	}
}

func valueSet(proc *exec.Process, p int32) {
	fmt.Printf("valueSet(%d)\n", p)
	obj, _ := LoadValue(proc, p+8)
	key := LoadString(proc, p+16)
	objs, _ := LoadValue(proc, p+8)

	fmt.Printf("Setting %s\n", key)

	if obj != nil {
		objVal := reflect.ValueOf(obj)
		fmt.Printf("Type: %s\n", objVal.Type().Name())
		if objVal.Type().Name() == "map[]" {
			fmt.Println("Settingfield")
			objVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(objs))
			return
		}

		switch v := objs.(type) {
		case string:
			objVal.FieldByName(key).SetString(v)
		case int64:
			objVal.FieldByName(key).SetInt(v)
		case float64:
			objVal.FieldByName(key).SetFloat(v)
		case bool:
			objVal.FieldByName(key).SetBool(v)
		case uint64:
			objVal.FieldByName(key).SetUint(v)
		default:
			fmt.Printf("Unknown mapped type: %s", reflect.TypeOf(obj).Name())
		}
	}

}

func valueDelete(proc *exec.Process, p int32) {
	fmt.Printf("valueDelete(%d)\n", p)
}

func valueIndex(proc *exec.Process, p int32) {
	fmt.Printf("valueIndex(%d)\n", p)
}

func valueSetIndex(proc *exec.Process, p int32) {
	fmt.Printf("valueSetIndex(%d)\n", p)
}

func valueCall(proc *exec.Process, p int32) {
	//fmt.Printf("valueCall(%d)\n", p)
	v, _ := LoadValue(proc, p+8)
	mV := LoadString(proc, p+16)
	args := LoadSliceOfValues(proc, p+32)

	fieldVal := reflect.ValueOf(v).MapIndex(reflect.ValueOf(mV))

	if fieldVal.IsValid() {
		fieldVal = reflect.ValueOf(fieldVal.Interface())
		//fmt.Printf("Calling %s from %+v with %+v\n", mV, v, args)
		//fmt.Printf("Field Val: %+v\n", fieldVal)

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
				e := r.(string)
				StoreValue(proc, int64(p+56), e)
				setUInt8(proc, p+64, 0)
			}
		}()

		argsVal := make([]reflect.Value, len(args))
		for i, v := range args {
			if v == nil {
				argsVal[i] = reflect.New(fieldVal.Type().In(i)).Elem()
			} else {
				argsVal[i] = reflect.ValueOf(v)
				// fmt.Printf("Arg %d: %s (%v)\n", i, argsVal[i].Type().String(), v)
			}
		}
		result := fieldVal.Call(argsVal)

		fmt.Printf("Result: %+v\n", result)
		setUInt8(proc, p+64, 1)
	}
}

func valueInvoke(proc *exec.Process, p int32) {
	fmt.Printf("valueInvoke(%d)\n", p)
}

func valueNew(proc *exec.Process, p int32) {
	//fmt.Printf("valueNew(%d)\n", p)

	lv, _ := LoadValue(proc, p+8)
	args := LoadSliceOfValues(proc, p+16)

	//fmt.Printf("%+v\n", lv)
	//fmt.Printf("%+v\n", args)

	copiedValue := reflect.New(reflect.TypeOf(lv)).Interface()

	switch v := copiedValue.(type) {
	case *uint8array:
		arrayLen := int(args[0].(float64))
		v.data = make([]byte, arrayLen)
		//fmt.Printf("Created new UInt8Array(%d)\n", arrayLen)
	}

	StoreValue(proc, int64(p+40), copiedValue)
	res := []byte{1}
	_, _ = proc.WriteAt(res, int64(p+48))
}

func valueLength(proc *exec.Process, p int32) {
	fmt.Printf("valueLength(%d)\n", p)
}

func valuePrepareString(proc *exec.Process, p int32) {
	fmt.Printf("valuePrepareString(%d)\n", p)
}

func valueLoadString(proc *exec.Process, p int32) {
	fmt.Printf("valueLoadString(%d)\n", p)
}

func valueInstanceOf(proc *exec.Process, p int32) {
	fmt.Printf("valueInstanceOf(%d)\n", p)
}

func copyBytesToGo(proc *exec.Process, p int32) {
	fmt.Printf("copyBytesToGo(%d)\n", p)
}

func copyBytesToJS(proc *exec.Process, p int32) {
	// fmt.Printf("copyBytesToJS(%d)\n", p)

	dst, _ := LoadValue(proc, p+8)
	src := LoadSlice(proc, p+16)

	if dstU8, ok := dst.(*uint8array); ok {
		copy(dstU8.data, src)
		setUInt64(proc, p+40, uint64(len(dstU8.data)))
		setUInt8(proc, p+48, 1)
		return
	}

	setUInt8(proc, p+48, 0)
}

var funcNames = []string{
	"runtime.wasmExit",
	"runtime.wasmWrite",
	"runtime.resetMemoryDataView",
	"runtime.nanotime1",
	"runtime.walltime1",
	"runtime.scheduleTimeoutEvent",
	"runtime.clearTimeoutEvent",
	"runtime.getRandomData",
	"syscall/js.finalizeRef",
	"syscall/js.stringVal",
	"syscall/js.valueGet",
	"syscall/js.valueSet",
	"syscall/js.valueDelete",
	"syscall/js.valueIndex",
	"syscall/js.valueSetIndex",
	"syscall/js.valueCall",
	"syscall/js.valueInvoke",
	"syscall/js.valueNew",
	"syscall/js.valueLength",
	"syscall/js.valuePrepareString",
	"syscall/js.valueLoadString",
	"syscall/js.valueInstanceOf",
	"syscall/js.copyBytesToGo",
	"syscall/js.copyBytesToJS",
	"debug",
}

var funcs = map[string]GoHostFunc{
	"runtime.wasmExit":              wasmExit,
	"runtime.wasmWrite":             wasmWrite,
	"runtime.resetMemoryDataView":   resetMemoryDataView,
	"runtime.nanotime1":             nanotime1,
	"runtime.walltime1":             walltime1,
	"runtime.scheduleTimeoutEvent":  scheduleTimeoutEvent,
	"runtime.clearTimeoutEvent":     clearTimeoutEvent,
	"runtime.getRandomData":         getRandomData,
	"syscall/js.finalizeRef":        finalizeRef,
	"syscall/js.stringVal":          stringVal,
	"syscall/js.valueGet":           valueGet,
	"syscall/js.valueSet":           valueSet,
	"syscall/js.valueDelete":        valueDelete,
	"syscall/js.valueIndex":         valueIndex,
	"syscall/js.valueSetIndex":      valueSetIndex,
	"syscall/js.valueCall":          valueCall,
	"syscall/js.valueInvoke":        valueInvoke,
	"syscall/js.valueNew":           valueNew,
	"syscall/js.valueLength":        valueLength,
	"syscall/js.valuePrepareString": valuePrepareString,
	"syscall/js.valueLoadString":    valueLoadString,
	"syscall/js.valueInstanceOf":    valueInstanceOf,
	"syscall/js.copyBytesToGo":      copyBytesToGo,
	"syscall/js.copyBytesToJS":      copyBytesToJS,
	"debug":                         debug,
}
