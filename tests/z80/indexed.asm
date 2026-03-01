.segment "CODE"

    ld ix,$1234
    ld iy,$5678
    ld a,(ix+5)
    ld (iy-2),a
    bit 3,(ix+5)
    bit 2,(iy-1)
    jp (ix)
    jp (iy)
    im 1
    rst $38
