.segment "CODE"

disp_hi = 127

    ld a,(ix-128)
    ld (iy+disp_hi),a
    bit 0,(ix+disp_hi)
    res 7,(iy-128)
    set 3,(ix-1)
    ret
