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
  image("tiny.nes.png", width: 50%),
  caption: [#link("https://en.wikipedia.org/wiki/Balloon_Fight")[Balloon Fight] running on #link("https://github.com/0xmukesh/tiny.nes")[tiny.nes]],
)

We wanted to build another emulator for the Game Boy, but this time we didn't want to build just another generic interpreter. Around then, we came across a blog post by Andrew Kelley titled #link("https://andrewkelley.me/post/jamulator.html")["Statically Recompiling NES Games into Native Executables with LLVM and Go"], and that's where the idea for Box Man came from.

Box Man is a tool that takes in Game Boy ROMs and emits LLVM IR, which can then be compiled into a native executable using tools like `clang`. The resulting binary plays that specific game natively, without needing an external Game Boy emulator at runtime. It's basically equivalent to ahead-of-time compilation of Game Boy ROMs.
