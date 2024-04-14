package assembler

import (
	"bytes"
	"hash/crc32"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/assembler/config"
	"github.com/retroenv/retrogolib/assert"
)

var ca65BlueTestCode = `
; PRG CRC32 checksum: d9730ffd
; CHR CRC32 checksum: d8f49994
; Overall CRC32 checksum: 79de3906
; Code base address: $8000

.setcpu "6502x"
.segment "HEADER"

.byte "NES", $1a                 ; Magic string that always begins an iNES header
.byte $02                        ; Number of 16KB PRG-ROM banks
.byte $01                        ; Number of 8KB CHR-ROM banks
.byte $01                        ; Control bits 1
.byte $00                        ; Control bits 2
.byte $00                        ; Number of 8KB PRG-RAM banks
.byte $00                        ; Video format NTSC/PAL

APU_DMC_FREQ = $4010
APU_FRAME = $4017
APU_SND_CHN = $4015
JOYPAD2 = $4017
PPU_ADDR = $2006
PPU_CTRL = $2000
PPU_DATA = $2007
PPU_MASK = $2001
PPU_STATUS = $2002


_var_0000_indexed = $0000
_var_000a = $000A
_var_0100_indexed = $0100
_var_0200_indexed = $0200
_var_0300_indexed = $0300
_var_0400_indexed = $0400
_var_0500_indexed = $0500
_var_0600_indexed = $0600
_var_0700_indexed = $0700


.segment "CODE"

Reset:
  sei                            ; $8000  78
  cld                            ; $8001  D8
  ldx #$FF                       ; $8002  A2 FF
  txs                            ; $8004  9A
  inx                            ; $8005  E8
  stx PPU_MASK                   ; $8006  8E 01 20
  stx APU_DMC_FREQ               ; $8009  8E 10 40
  stx PPU_CTRL                   ; $800C  8E 00 20
  bit PPU_STATUS                 ; $800F  2C 02 20
  bit APU_SND_CHN                ; $8012  2C 15 40
  lda #$40                       ; $8015  A9 40
  sta APU_FRAME                  ; $8017  8D 17 40
  lda #$0F                       ; $801A  A9 0F
  sta APU_SND_CHN                ; $801C  8D 15 40
  jsr _func_8047                 ; $801F  20 47 80
  jsr _func_804d                 ; $8022  20 4D 80
  jsr _func_8047                 ; $8025  20 47 80
  jsr _func_8072                 ; $8028  20 72 80
  ldx PPU_STATUS                 ; $802B  AE 02 20
  ldx #$3F                       ; $802E  A2 3F
  stx PPU_ADDR                   ; $8030  8E 06 20
  ldx #$00                       ; $8033  A2 00
  stx PPU_ADDR                   ; $8035  8E 06 20
  lda a:_var_000a                ; $8038  AD 0A 00
  sta PPU_DATA                   ; $803B  8D 07 20
  lda #$1E                       ; $803E  A9 1E
  sta PPU_MASK                   ; $8040  8D 01 20

_label_8043:
  nop                            ; $8043  EA
  jmp _label_8043                ; $8044  4C 43 80

_func_8047:
  bit PPU_STATUS                 ; $8047  2C 02 20
  bpl _func_8047                 ; $804A  10 FB
  rts                            ; $804C  60

_func_804d:
  lda #$00                       ; $804D  A9 00
  tax                            ; $804F  AA

_label_8050:
  sta z:_var_0000_indexed,X      ; $8050  95 00
  cpx #$FE                       ; $8052  E0 FE
  bcs _label_8059                ; $8054  B0 03
  sta a:_var_0100_indexed,X      ; $8056  9D 00 01

_label_8059:
  sta a:_var_0200_indexed,X      ; $8059  9D 00 02
  sta a:_var_0300_indexed,X      ; $805C  9D 00 03
  sta a:_var_0400_indexed,X      ; $805F  9D 00 04
  sta a:_var_0500_indexed,X      ; $8062  9D 00 05
  sta a:_var_0600_indexed,X      ; $8065  9D 00 06
  sta a:_var_0700_indexed,X      ; $8068  9D 00 07
  inx                            ; $806B  E8
  bne _label_8071                ; $806C  D0 03
  jmp _label_8050                ; $806E  4C 50 80

_label_8071:
  rts                            ; $8071  60

_func_8072:
  lda #$11                       ; $8072  A9 11
  sta a:_var_000a                ; $8074  8D 0A 00
  rts                            ; $8077  60

.segment "TILES"


.segment "VECTORS"

.addr 0, Reset, 0
`

var ca65BasicConfig = `
MEMORY {
    ZP:     start = $00,    size = $100,    type = rw, file = "";
    RAM:    start = $0200,  size = $600,    type = rw, file = "";
    HDR:    start = $0000,  size = $10,     type = ro, file = %O, fill = yes;
    PRG:    start = $8000,  size = $8000,   type = ro, file = %O, fill = yes;
    CHR:    start = $0000,  size = $2000,   type = ro, file = %O, fill = yes;
}

SEGMENTS {
    ZEROPAGE:   load = ZP,  type = zp;
    OAM:        load = RAM, type = bss, start = $200, optional = yes;
    BSS:        load = RAM, type = bss;
    HEADER:     load = HDR, type = ro;
    CODE:       load = PRG, type = ro, start = $8000;
    DPCM:       load = PRG, type = ro, start = $C000, optional = yes;
    VECTORS:    load = PRG, type = ro, start = $FFFA;
    TILES:      load = CHR, type = ro;
}
`

func TestAssemblerCa65BlueExample(t *testing.T) {
	cfg := &config.Config{}
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(ca65BasicConfig)))
	cfg.Arch = arch.NewNES()

	reader := strings.NewReader(ca65BlueTestCode)
	var buf bytes.Buffer
	asm := New(cfg, reader, &buf)

	assert.NoError(t, asm.Process())
	assert.Equal(t, 40976, buf.Len())

	crc32q := crc32.MakeTable(crc32.IEEE)
	b := buf.Bytes()
	sum := crc32.Checksum(b[16:], crc32q) // skip header for checksum
	assert.Equal(t, 0x79de3906, sum)
}

var unitTestConfig = `
MEMORY {
    HDR:    start = $0000,  size = $100,     type = ro;
}

SEGMENTS {
    HEADER:     load = HDR, type = ro;
}
`
