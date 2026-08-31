SECTION "Header", ROM0[$0100]
    nop
    jp Start

SECTION "Nintendo Logo", ROM0[$0104]
    ds $30, 0

SECTION "Cartridge Header", ROM0[$0134]
    db "test"
    ds $0B, 0
    db $00 ; CGB flag
    dw $0000 ; New licensee
    db $00 ; SGB flag
    db $00 ; Cart type (ROM ONLY)
    db $00 ; ROM size (32 KB)
    db $00 ; RAM size
    db $00 ; Destination
    db $00 ; Old licensee
    db $00 ; Version
    db $00 ; Header checksum
    dw $0000 ; Global checksum

SECTION "Game Code", ROM0[$0150]
Start:
    ld b, $03
    ld c, $02
    ld e, $01
    ld h, $c0
    ld l, $00
    ld b, $42
    ld [hl], b
    ld a, [hl]

SECTION "ROM Pad Bank 0", ROM0[$015E]
    ds $3EA2, 0

SECTION "ROM Pad Bank 1", ROMX, BANK[1]
    ds $4000, 0
