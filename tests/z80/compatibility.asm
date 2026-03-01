.segment "CODE"

start:
    jr nz,near
    jp done
near:
    ld a,(table+1)
    djnz start
done:
    ret

table:
    .byte $10,$20,$30
