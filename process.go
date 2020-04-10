package main

import "fmt"

var importProcess = map[string]interface{}{
	"getuid":    func() { fmt.Println("getuid") },
	"getgid":    func() { fmt.Println("getgid") },
	"geteuid":   func() { fmt.Println("geteuid") },
	"getegid":   func() { fmt.Println("getegid") },
	"getgroups": func() { fmt.Println("getgroups") },
	"pid":       -1,
	"ppid":      -1,
	"umask":     func() { fmt.Println("umask") },
	"cwd":       func() { fmt.Println("cwd") },
	"chdir":     func() { fmt.Println("chdir") },
}
