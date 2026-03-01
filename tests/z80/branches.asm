.segment "CODE"

start:
    jr nz,skip
    nop
skip:
    jr start
    jp nz,target
    call target
target:
    ret
