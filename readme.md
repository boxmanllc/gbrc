# gbrc

gbrc is an LLVM-based static recompiler for the Game Boy DMG-01. With gbrc, you can recompile Game Boy games for any system and build your own frontend using the runtime library. As of right now, gbrc only supports games which don't require MBC (memory bank controller).

## Installation

```bash
go install github.com/boxmanllc/gbrc
```

## Trying out examples

We use [baker](https://github.com/rv178/baker) as our build system, because why not?

All the frontend examples are present in [`runtime/examples`](./runtime/examples). Currently, it has a desktop frontend (using SDL2) and browser frontend (using Emscripten).

### Desktop

It needs `clang`, `opt` and SDL2 dev headers to be present in your `PATH`.

```bash
bake sdl
FRONTEND=sdl bake build
./build/tetris
```

### Browser

It needs Emscripten SDK to be present in your `PATH`.

````bash
bake web
FRONTEND=web bake build
python3 -m http.server -d ./build
```-->
````
