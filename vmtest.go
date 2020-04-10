package main

import (
	"encoding/binary"
	"fmt"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/validate"
	"github.com/go-interpreter/wagon/wasm"
	"log"
	"os"
	"reflect"
)

func main() {
	f, err := os.Open("./app/main.wasm")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fmt.Printf("Reading module\n")
	m, err := wasm.ReadModule(f, importer)
	if err != nil {
		log.Fatal(err)
	}
	//err = validate.VerifyModule(m)
	//if err != nil {
	//	log.Fatal(err)
	//}

	fmt.Println("Creating VM")

	vm, err := exec.NewVM(m)
	if err != nil {
		log.Fatalf("could not create VM: %v", err)
	}
	run, ok := m.Export.Entries["run"]

	if !ok {
		panic("cannot find run function")
	}

	scope["_resume"] = func() {
		fmt.Printf("Called _resume\n")
		if scope["exited"].(bool) {
			panic("Go program has already exited")
		}
		run, ok := m.Export.Entries["resume"]

		if !ok {
			panic("Cannot find resume function in wasm")
		}

		ret, err := vm.ExecCode(int64(run.Index))

		if err != nil {
			panic(err)
		}

		fmt.Printf("Return: %+v\n", ret)

		event := scope["_pendingEvent"].(wasmEvent)
		event.Result = []interface{}{ret}
		scope["_pendingEvent"] = event
	}

	memory := vm.Memory()
	offset := 4096

	strPtr := func(str string) int32 {
		ptr := offset
		bytes := append([]byte(str), 0x00)

		for i := 0; i < len(bytes); i++ {
			memory[offset+i] = bytes[i]
		}

		offset += len(bytes)
		if offset%8 != 0 {
			offset += 8 - (offset % 8)
		}
		return int32(ptr)
	}

	argv := []string{"js"}
	argc := len(argv)
	argvPtrs := []int32{}

	for _, v := range argv {
		argvPtrs = append(argvPtrs, strPtr(v))
	}
	argvPtrs = append(argvPtrs, 0)

	argvPtr := offset
	for _, v := range argvPtrs {
		binary.LittleEndian.PutUint64(memory[offset:], uint64(v))
		offset += 8
	}

	o, err := vm.ExecCode(int64(run.Index), uint64(argc), uint64(argvPtr))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%[1]v (%[1]T)\n", o)

}

func importer(name string) (*wasm.Module, error) {
	fmt.Printf("Loading module %s\n", name)

	if name == "go" {
		m := wasm.NewModule()

		sections := []wasm.FunctionSig{}
		indexSpace := []wasm.Function{}
		exports := map[string]wasm.ExportEntry{}

		for i, v := range funcNames {
			sections = append(sections, wasm.FunctionSig{
				Form:        uint8(i),
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{},
			})
			indexSpace = append(indexSpace, wasm.Function{
				Sig:  &sections[i],
				Host: reflect.ValueOf(funcs[v]),
				Body: &wasm.FunctionBody{},
			})
			exports[v] = wasm.ExportEntry{
				FieldStr: v,
				Kind:     wasm.ExternalFunction,
				Index:    uint32(i),
			}
		}

		m.Types = &wasm.SectionTypes{
			// List of all function types available in this module.
			// There is only one: (func [int32] -> [int32])
			Entries: sections,
		}
		m.FunctionIndexSpace = indexSpace

		m.Export = &wasm.SectionExports{
			Entries: exports,
		}
		fmt.Println("module created")
		return m, nil
	}

	f, err := os.Open(name + ".wasm")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := wasm.ReadModule(f, nil)
	if err != nil {
		return nil, err
	}
	err = validate.VerifyModule(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
