.segment "CODE"

start:
    jp target+1
    ld a,(data+1)
    ld (data+2),a
    in a,($10+1)
    out ($20+2),a
target:
    nop
    nop
data:
    .byte $11,$22,$33,$44
