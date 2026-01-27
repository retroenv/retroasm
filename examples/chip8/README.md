# Chip-8 Assembly Examples

This directory contains example programs written in Chip-8 assembly language.

## hello.asm - Simple Sprite Display

A minimal program that displays a sprite on screen.

**Features:**
- Basic sprite drawing
- Simple infinite loop pattern
- Hex data with `.byte` directive

**How to assemble:**
```bash
retroasm -cpu chip8 -o hello.ch8 examples/chip8/hello.asm
```

**Running the program:**
Use any Chip-8 emulator to run the `hello.ch8` file. It will display the digit "0" sprite on screen.

## cube.asm - Companion Cube Animation

A more complex program that animates a sprite bouncing horizontally across the screen.

**Source:** Adapted from [wernsey/chip8](https://github.com/wernsey/chip8/blob/master/examples/cube.asm)

**Features:**
- Uses register aliases for readable code
- Demonstrates sprite drawing with DRW instruction
- Shows conditional jumps with SE/SNE instructions
- Includes binary sprite data using `.db` directive

**How to assemble:**
```bash
retroasm -cpu chip8 -o cube.ch8 examples/chip8/cube.asm
```

**Running the program:**
Use any Chip-8 emulator to run the `cube.ch8` file. The companion cube will bounce back and forth across the screen.

## Chip-8 Assembly Language Features

### Register Aliases
Instead of using register names directly, you can define aliases:
```asm
boxx = 0  ; V0
boxy = 1  ; V1
```

Then use the alias in instructions:
```asm
ld v0, 10    ; Using register directly
; or
ld v1, 20    ; V1 is boxy
```

### Constants
Define constants using `=`:
```asm
SCREEN_WIDTH = 64
SCREEN_HEIGHT = 32
```

### Binary Literals
Use `%` prefix for binary numbers:
```asm
.db %11110000  ; Binary: 0xF0
```

### Sprite Data
Define sprite data using `.db` directive:
```asm
sprite:
    .db %01111110
    .db %10000001
    .db %10000001
    .db %01111110
```

## Common Chip-8 Instructions

| Instruction | Description |
|-------------|-------------|
| `CLS` | Clear screen |
| `RET` | Return from subroutine |
| `JP addr` | Jump to address |
| `CALL addr` | Call subroutine |
| `SE Vx, byte` | Skip if Vx == byte |
| `SNE Vx, byte` | Skip if Vx != byte |
| `LD Vx, byte` | Load byte into Vx |
| `ADD Vx, byte` | Add byte to Vx |
| `DRW Vx, Vy, nibble` | Draw sprite at (Vx, Vy) |

For a complete instruction reference, see the [Chip-8 specification](http://devernay.free.fr/hacks/chip8/C8TECH10.HTM).
