.segment "CODE"

delta = 2
disp = 1

start:
    jp target+delta
    ld a,(table+disp)
    ld (table+disp+1),a
    ld a,(ix+disp)
target:
    nop
table:
    .byte $10,$20,$30
