package linker

import (
	"debug/elf"
	"math"
	"math/bits"
	"rvld/pkg/utils"
)

type InputSection struct {
	File     *ObjectFile
	Contents []byte
	Shndx    uint32
	ShSize   uint32
	IsAlive  bool
	P2Align  uint8
	Offset   uint32

	OutputSection *OutputSection

	RelsecInx uint32
	Rels      []Rela
}

func NewInputSection(ctx *Context, name string, file *ObjectFile, shndx uint32) *InputSection {
	s := &InputSection{
		File:      file,
		Shndx:     shndx,
		IsAlive:   true,
		Offset:    math.MaxUint32,
		RelsecInx: math.MaxUint32,
		ShSize:    math.MaxUint32,
	}

	shdr := s.Shdr()
	s.Contents = file.File.Contents[shdr.Offset : shdr.Offset+shdr.Size]

	// If section's flag is SHF_COMPRESSED, skip it
	utils.Assert(shdr.Flags&uint64(elf.SHF_COMPRESSED) == 0)
	s.ShSize = uint32(shdr.Size)

	// Calculate trailing zeros of align
	ToP2Align := func(align uint64) uint8 {
		if align == 0 {
			return 0
		}

		return uint8(bits.TrailingZeros64(align))
	}
	s.P2Align = ToP2Align(shdr.Addralign)

	s.OutputSection = GetOutputSection(ctx, name, shdr.Type, shdr.Flags)

	return s
}

func (i *InputSection) Shdr() *SectionHeader {
	utils.Assert(i.Shndx < uint32(len(i.File.InputFile.Sections)))
	return &i.File.InputFile.Sections[i.Shndx]
}

func (i *InputSection) Name() string {
	return GetNameFromTable(i.File.StrTable, i.Shdr().Name)
}

func (i *InputSection) WriteTo(ctx *Context, buf []byte) {
	if i.Shdr().Type == uint32(elf.SHT_NOBITS) || i.ShSize == 0 {
		return
	}

	i.CopyContents(buf)

	if i.Shdr().Flags&uint64(elf.SHF_ALLOC) != 0 {
		i.ApplyRelocAlloc(ctx, buf)
	}
}

func (i *InputSection) CopyContents(buf []byte) {
	copy(buf, i.Contents)
}

func (i *InputSection) GetRels() []Rela {
	if i.RelsecInx == math.MaxUint32 || i.Rels != nil {
		return i.Rels
	}

	bs := i.File.GetBytesFromShdr(&i.File.InputFile.Sections[i.RelsecInx])
	i.Rels = utils.ReadSlice[Rela](bs, int(RelaSize))
	return i.Rels
}

func (i *InputSection) GetAddr() uint64 {
	return i.OutputSection.Shdr.Addr + uint64(i.Offset)
}

func (i *InputSection) ScanRelocations() {
	rels := i.GetRels()
	for _, rel := range rels {
		sym := i.File.Symbols[rel.Sym]
		if sym.File == nil {
			continue
		}

		if rel.Type == uint32(elf.R_RISCV_TLS_GOT_HI20) {
			sym.Flags |= NeedsGotTp
		}
	}
}

func (i *InputSection) ApplyRelocAlloc(ctx *Context, base []byte) {
	rels := i.GetRels()

	for _, rel := range rels {
		if rel.Type == uint32(elf.R_RISCV_NONE) || rel.Type == uint32(elf.R_RISCV_RELAX) {
			continue
		}

		sym := i.File.Symbols[rel.Sym]
		loc := base[rel.Offset:]

		if sym.File == nil {
			continue
		}

		S := sym.GetAddr()
		A := uint64(rel.Addend)
		P := i.GetAddr() + rel.Offset

		switch elf.R_RISCV(rel.Type) {
		case elf.R_RISCV_32:
			utils.Write(loc, uint32(S+A))
		case elf.R_RISCV_64:
			utils.Write(loc, S+A)
		case elf.R_RISCV_BRANCH:
			WriteBtype(loc, uint32(S+A-P))
		case elf.R_RISCV_JAL:
			WriteJtype(loc, uint32(S+A-P))
		case elf.R_RISCV_CALL, elf.R_RISCV_CALL_PLT:
			val := uint32(S + A - P)
			WriteUtype(loc, val)
			WriteItype(loc[4:], val)
		case elf.R_RISCV_TLS_GOT_HI20:
			utils.Write(loc, uint32(sym.GetGotTpAddr(ctx)+A-P))
		case elf.R_RISCV_PCREL_HI20:
			utils.Write(loc, uint32(S+A-P))
		case elf.R_RISCV_HI20:
			WriteUtype(loc, uint32(S+A))
		case elf.R_RISCV_LO12_I, elf.R_RISCV_LO12_S:
			val := S + A
			if rel.Type == uint32(elf.R_RISCV_LO12_I) {
				WriteItype(loc, uint32(val))
			} else {
				WriteStype(loc, uint32(val))
			}

			if utils.SignExtend(val, 11) == val {
				SetRs1(loc, 0)
			}
		case elf.R_RISCV_TPREL_LO12_I, elf.R_RISCV_TPREL_LO12_S:
			val := S + A - ctx.TpAddr
			if rel.Type == uint32(elf.R_RISCV_TPREL_LO12_I) {
				WriteItype(loc, uint32(val))
			} else {
				WriteStype(loc, uint32(val))
			}

			if utils.SignExtend(val, 11) == val {
				SetRs1(loc, 4)
			}
		}
	}

	for a := 0; a < len(rels); a++ {
		switch elf.R_RISCV(rels[a].Type) {
		case elf.R_RISCV_PCREL_LO12_I, elf.R_RISCV_PCREL_LO12_S:
			sym := i.File.Symbols[rels[a].Sym]
			utils.Assert(sym.InputSection == i)
			loc := base[rels[a].Offset:]
			val := utils.Read[uint32](base[sym.Value:])

			if rels[a].Type == uint32(elf.R_RISCV_PCREL_LO12_I) {
				WriteItype(loc, val)
			} else {
				WriteStype(loc, val)
			}
		}
	}

	for a := 0; a < len(rels); a++ {
		switch elf.R_RISCV(rels[a].Type) {
		case elf.R_RISCV_PCREL_HI20, elf.R_RISCV_TLS_GOT_HI20:
			loc := base[rels[a].Offset:]
			val := utils.Read[uint32](loc)

			utils.Write(loc, utils.Read[uint32](i.Contents[rels[a].Offset:]))
			WriteUtype(loc, val)
		}
	}
}

// github.com/jameslzhu/riscv-card/riscv-card.pdf
func itype(val uint32) uint32 {
	return val << 20
}

func stype(val uint32) uint32 {
	return utils.Bits(val, 11, 5)<<25 | utils.Bits(val, 4, 0)<<7
}

func btype(val uint32) uint32 {
	return utils.Bit(val, 12)<<31 | utils.Bits(val, 10, 5)<<25 |
		utils.Bits(val, 4, 1)<<8 | utils.Bit(val, 11)<<7
}

func utype(val uint32) uint32 {
	return (val + 0x800) & 0xffff_f000
}

func jtype(val uint32) uint32 {
	return utils.Bit(val, 20)<<31 | utils.Bits(val, 10, 1)<<21 |
		utils.Bit(val, 11)<<20 | utils.Bits(val, 19, 12)<<12
}

func cbtype(val uint16) uint16 {
	return utils.Bit(val, 8)<<12 | utils.Bit(val, 4)<<11 | utils.Bit(val, 3)<<10 |
		utils.Bit(val, 7)<<6 | utils.Bit(val, 6)<<5 | utils.Bit(val, 2)<<4 |
		utils.Bit(val, 1)<<3 | utils.Bit(val, 5)<<2
}

func cjtype(val uint16) uint16 {
	return utils.Bit(val, 11)<<12 | utils.Bit(val, 4)<<11 | utils.Bit(val, 9)<<10 |
		utils.Bit(val, 8)<<9 | utils.Bit(val, 10)<<8 | utils.Bit(val, 6)<<7 |
		utils.Bit(val, 7)<<6 | utils.Bit(val, 3)<<5 | utils.Bit(val, 2)<<4 |
		utils.Bit(val, 1)<<3 | utils.Bit(val, 5)<<2
}

func WriteItype(loc []byte, val uint32) {
	mask := uint32(0b000000_00000_11111_111_11111_1111111)
	utils.Write[uint32](loc, (utils.Read[uint32](loc)&mask)|itype(val))
}

func WriteStype(loc []byte, val uint32) {
	mask := uint32(0b000000_11111_11111_111_00000_1111111)
	utils.Write[uint32](loc, (utils.Read[uint32](loc)&mask)|stype(val))
}

func WriteBtype(loc []byte, val uint32) {
	mask := uint32(0b000000_11111_11111_111_00000_1111111)
	utils.Write[uint32](loc, (utils.Read[uint32](loc)&mask)|btype(val))
}

func WriteUtype(loc []byte, val uint32) {
	mask := uint32(0b000000_00000_00000_000_11111_1111111)
	utils.Write[uint32](loc, (utils.Read[uint32](loc)&mask)|utype(val))
}

func WriteJtype(loc []byte, val uint32) {
	mask := uint32(0b000000_00000_00000_000_11111_1111111)
	utils.Write[uint32](loc, (utils.Read[uint32](loc)&mask)|jtype(val))
}

func SetRs1(loc []byte, rs1 uint32) {
	utils.Write[uint32](loc, utils.Read[uint32](loc)&(0b111111_11111_00000_111_11111_1111111))
	utils.Write[uint32](loc, utils.Read[uint32](loc)|(rs1<<15))
}

