package main

import "embed"

//go:embed runtime/include/*.h runtime/include/hardware/*.h runtime/src/*.c runtime/src/hardware/*.c
var runtimeFS embed.FS
