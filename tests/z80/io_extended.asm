.segment "CODE"

    ld a,($1234)
    ld ($2345),a
    ld bc,($3456)
    ld ($4567),bc
    in a,($12)
    out ($34),a
    in b,(c)
    out (c),e
