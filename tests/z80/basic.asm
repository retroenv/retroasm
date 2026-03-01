.segment "CODE"

start:
    nop
    ld bc,$1234
    ld a,42
loop:
    bit 3,a
    jr nz,loop
