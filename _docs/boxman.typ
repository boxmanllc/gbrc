#import "@preview/fletcher:0.5.8" as fletcher: diagram, edge, node

#set page(numbering: "1")
#set heading(numbering: "1.a")

#show link: set text(
  fill: blue,
)
#show heading: set block(
  below: 1em,
)

#let full-date = datetime.today().display("[day] [month repr:long] [year]")

#align(center, text(17pt)[*Box Man: Static recompilation of Game Boy games with LLVM*])
#align(center)[
  #grid(
    columns: 2,
    column-gutter: 8pt,
    align: horizon,
    text(13pt)[*Team Gloog*], image("gloog.png", height: 1.5em),
  )
]
#align(center, text(11pt)[#full-date])

= Introduction

Building software-based emulators has always been an interesting exercise for recreational and hobbyist programmers. There is something satisfying about running old, nostalgic games like #link("https://en.wikipedia.org/wiki/The_Legend_of_Zelda")[The Legend of Zelda], #link("https://en.wikipedia.org/wiki/Super_Mario_Bros.")[Super Mario Bros.], or #link("https://en.wikipedia.org/wiki/Pok%C3%A9mon_Red%2C_Blue%2C_and_Yellow")[Pokémon Red/Blue/Yellow] through an emulator you built yourself.

Earlier this year, one of us got nerd-snipped into emulation development (commonly referred to as "emudev") by this site called #link("http://visual6502.org/JSSim/index.html")[Visual 6502], which is a browser-based transistor-level 6502 emulator. From then on, we have built a couple of small-scale emulators for old 2D-based retro game systems like #link("https://en.wikipedia.org/wiki/CHIP-8")[CHIP-8] and #link("https://en.wikipedia.org/wiki/Nintendo_Entertainment_System")[NES].

#figure(
  image("tiny.nes.png", width: 40%),
  caption: [#link("https://en.wikipedia.org/wiki/Balloon_Fight")[Balloon Fight] running on #link("https://github.com/0xmukesh/tiny.nes")[tiny.nes]],
)

We wanted to build another emulator for the Game Boy, but this time we didn't want to build just another generic interpreter. Around then, we came across a blog post by Andrew Kelley titled #link("https://andrewkelley.me/post/jamulator.html")["Statically Recompiling NES Games into Native Executables with LLVM and Go"], and that's where the idea for Box Man came from.

Within the reverse engineering ecosystem, _manual_ static recompilation of retro games is a very old and well-established practice. Some of the most popular community efforts in this context are #link("https://github.com/n64decomp/sm64")[Super Mario 64 decompilation] and #link("https://github.com/pret/pokered")[Pokémon Red/Blue disassembly]. The researchers had to painstakingly go through the binaries of these games, analyze them and turn them into readable and byte-matching source code. These community efforts helped in improving the documentation of these systems which led to rise to automated static recompiler tools like #link("https://github.com/N64Recomp/N64Recomp")[N64Recomp] which is being used by #link("https://github.com/Zelda64Recomp/Zelda64Recomp")[Zeld64Recomp] project to statically recompile #link("https://en.wikipedia.org/wiki/The_Legend_of_Zelda%3A_Majora's_Mask")[The Legend of Zelda: Majora's Mask] game.

Box Man is a tool that takes in Game Boy ROMs and emits LLVM IR, which can then be compiled into a native executable using tools like `clang`. The resulting binary plays that specific game natively, without needing an external Game Boy emulator at runtime. It's basically equivalent to ahead-of-time compilation of Game Boy ROMs.

= Architecture

Box Man isn't a fully self-contained static recompiler. It focuses mainly on statically recompiling the CPU, i.e., the actual instruction stream the game executes. Hardware peripherals like the PPU, APU, and timer are not part of the instruction stream, and the CPU communicates with them through memory-mapped I/O. Due to this, we implement the behavior of the above hardware peripherals using a small, hand-written runtime library, which is linked at compile time alongside the recompiled CPU code.

Firstly, the ROM's file header is parsed to fetch the metadata such as the game's title, ROM and RAM sizes, and additional checksums for integrity. After that, the control-flow analysis is done to recover the structure of the game's code. Starting from known entry points, which include the cartridge's boot address and the fixed interrupt vector addresses, Box Man recursively walks through the code, splitting into different blocks at every branch, call, and return. Anything which wasn't encountered by this recursive walk is treated as data rather than code. This differentiation in data and code helps in figuring out whether real instructions are present and where graphics data is present.

After the basic blocks are generated, each one of them is lowered into LLVM IR, where the entire game loop is represented as a single function. Instructions that try to target hardware peripheral specific addresses are interrupted by the runtime libraries to tasks such as rendering the graphics, taking the input from the keyboard, playing the sound, etc.

#figure(
  diagram(
    node-stroke: 0.6pt,
    node-corner-radius: 4pt,
    node-inset: 10pt,
    spacing: (3em, 2em),

    node((0, 0), [`game.gb`]),
    edge((0, 0), (0, 1), "-|>"),

    node((0, 1), [ROM loader]),
    edge((0, 1), (0, 2), "-|>"),

    node((0, 2), [*Decoder*: decodes opcodes into a descriptor table]),
    edge((0, 2), (0, 3), "-|>"),

    node((0, 3), [*Analyzer*: Builds CFG, separates code from data]),
    edge((0, 3), (0, 4), "-|>"),

    node((0, 4), [LLVM IR generator]),
    edge((0, 4), (0, 5), "-|>"),

    node((0, 5), [`game.ll`]),
    edge((0, 5), (0, 6), "-|>", text(size: 9pt)[clang -c], label-side: right),

    node((0, 6), [`game.o`]),

    node((0.7, 6), [*Runtime libraries* \ PPU, Joypad, Timer, Audio, Memory], inset: 8pt),
    edge((0.7, 6), (0, 6), "-|>"),

    edge((0, 6), (0, 7), "-|>", text(size: 9pt)[link], label-side: right),
    node((0, 7), [`game`: Native executable]),
  ),
  caption: [Box Man's static recompilation pipeline],
)
