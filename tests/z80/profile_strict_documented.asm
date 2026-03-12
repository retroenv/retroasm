.segment "CODE"

start:
    nop
    bit 3,a
    jr nz,start
