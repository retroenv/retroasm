; Companion Cube Sample program
; Adapted from: https://github.com/wernsey/chip8/blob/master/examples/cube.asm
;
; This program animates a sprite moving horizontally across the screen,
; bouncing between the left and right edges.

; Register aliases using = assignment
boxx = 0  ; V0 - current X position
boxy = 1  ; V1 - current Y position
oldx = 2  ; V2 - previous X position
oldy = 3  ; V3 - previous Y position
dirx = 4  ; V4 - direction (0=left, 1=right)
diry = 5  ; V5 - Y direction (unused)
tmp = 14  ; VE - temporary value

; Start of program
cls

; Initialize positions and direction
ld  v0, 1        ; boxx = 1
ld  v4, 1        ; dirx = 1 (moving right)
ld  v1, 10       ; boxy = 10
ld  i, sprite1   ; load sprite address
drw v0, v1, 8    ; draw sprite at initial position
ld  ve, 1        ; tmp = 1

loop:
	ld v2, v0        ; oldx = boxx
	ld v3, v1        ; oldy = boxy

	se v4, 0         ; skip if dirx == 0
	jp sub1

	add v0, 1        ; boxx++ (moving right)

	sne v0, 56       ; skip if boxx != 56
	ld  v4, 1        ; hit right edge, set dirx = 1 (go left)
	jp draw1
sub1:
	sub v0, ve       ; boxx-- (moving left)

	sne v0, 0        ; skip if boxx != 0
	ld  v4, 0        ; hit left edge, set dirx = 0 (go right)

draw1:
	ld  i, sprite1   ; load sprite address
	drw v2, v3, 8    ; erase old sprite (XOR)
	drw v0, v1, 8    ; draw new sprite

	jp  loop         ; infinite loop

; Companion cube sprite (8x8 pixels)
sprite1:
	.db  %01111110
	.db  %10000001
	.db  %10100101
	.db  %10111101
	.db  %10111101
	.db  %10011001
	.db  %10000001
	.db  %01111110
