package main

import "embed"

//go:embed runtime/include/*.h runtime/src/*.c
var runtimeFS embed.FS
