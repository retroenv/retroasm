; Simple Chip-8 program example
; Clears the screen and displays a simple pattern

start:
    cls                   ; Clear the screen
    ld v0, 5              ; Load 5 into V0 (X coordinate)
    ld v1, 10             ; Load 10 into V1 (Y coordinate)
    ld i, sprite_data     ; Load sprite address into I
    drw v0, v1, 5         ; Draw sprite at (V0, V1) with height 5

loop:
    jp loop               ; Infinite loop

sprite_data:
    .byte $F0, $90, $90, $90, $F0  ; Simple sprite pattern (digit "0")
