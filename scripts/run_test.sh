#!/usr/bin/bash

rgbasm -o ./build/test.o ./tests/test.asm
rgblink -o ./build/test.gb ./build/test.o
go run . --rom=./build/test.gb --debug
lli ./build/test.ll
