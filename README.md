# WASM-VM

Wagon Experiment Simulating GOOS=js


The idea is to run a golang wasm application (see app dir) inside a Wagon VM.

To do that, I implemented the wasm_exec from GOOS=js in Golang considering the wagon caracteristics.
The code is very bad and incomplete, but it does print a Hello World.
