package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

var constants = map[string]interface{}{
	"O_WRONLY": -1,
	"O_RDWR":   -1,
	"O_CREAT":  -1,
	"O_TRUNC":  -1,
	"O_APPEND": -1,
	"O_EXCL":   -1,
}

var fsImport = map[string]interface{}{
	"write": func(fd float64, buf *uint8array, offset int, _length float64, position interface{}, callback interface{}) {
		var w io.Writer
		length := int(_length)
		switch fd {
		case float64(syscall.Stdout):
			w = os.Stdout
		case float64(syscall.Stderr):
			w = os.Stderr
		case float64(syscall.Stdin):
			w = os.Stdin
		default:
			fmt.Printf("Invalid FD: %f\n", fd)
			return
		}
		_, _ = fmt.Fprintf(w, "WASM: %s", string(buf.data[offset:offset+length]))
	}, // (fd, buf, offset, length, position, callback) {},,
	"chmod":     func() { fmt.Println("chmod") },     // (path, mode, callback) { callback(enosys()); },,
	"chown":     func() { fmt.Println("chown") },     // (path, uid, gid, callback) { callback(enosys()); },,
	"close":     func() { fmt.Println("close") },     // (fd, callback) { callback(enosys()); },,
	"fchmod":    func() { fmt.Println("fchmod") },    // (fd, mode, callback) { callback(enosys()); },,
	"fchown":    func() { fmt.Println("fchown") },    // (fd, uid, gid, callback) { callback(enosys()); },,
	"fstat":     func() { fmt.Println("fstat") },     // (fd, callback) { callback(enosys()); },,
	"fsync":     func() { fmt.Println("fsync") },     // (fd, callback) { callback(null); },,
	"ftruncate": func() { fmt.Println("ftruncate") }, // (fd, length, callback) { callback(enosys()); },,
	"lchown":    func() { fmt.Println("lchown") },    // (path, uid, gid, callback) { callback(enosys()); },,
	"link":      func() { fmt.Println("link") },      // (path, link, callback) { callback(enosys()); },,
	"lstat":     func() { fmt.Println("lstat") },     // (path, callback) { callback(enosys()); },,
	"mkdir":     func() { fmt.Println("mkdir") },     // (path, perm, callback) { callback(enosys()); },,
	"open":      func() { fmt.Println("open") },      // (path, flags, mode, callback) { callback(enosys()); },,
	"read":      func() { fmt.Println("read") },      // (fd, buffer, offset, length, position, callback) { callback(enosys()); },,
	"readdir":   func() { fmt.Println("readdir") },   // (path, callback) { callback(enosys()); },,
	"readlink":  func() { fmt.Println("readlink") },  // (path, callback) { callback(enosys()); },,
	"rename":    func() { fmt.Println("rename") },    // (from, to, callback) { callback(enosys()); },,
	"rmdir":     func() { fmt.Println("rmdir") },     // (path, callback) { callback(enosys()); },,
	"stat":      func() { fmt.Println("stat") },      // (path, callback) { callback(enosys()); },,
	"symlink":   func() { fmt.Println("symlink") },   // (path, link, callback) { callback(enosys()); },,
	"truncate":  func() { fmt.Println("truncate") },  // (path, length, callback) { callback(enosys()); },,
	"unlink":    func() { fmt.Println("unlink") },    // (path, callback) { callback(enosys()); },,
	"utimes":    func() { fmt.Println("utimes") },    // (path, atime, mtime, callback) { callback(enosys()); },,
	"constants": constants,
}
