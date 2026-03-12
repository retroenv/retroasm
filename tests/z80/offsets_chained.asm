.segment "CODE"

start:
    jp target+2-1
    ld a,(data+3-1)
    ld (data+4-1),a
    in a,($10+3-1)
    out ($20+2-1),a
target:
    nop
data:
    .byte $11,$22,$33,$44,$55
