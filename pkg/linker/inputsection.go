package linker

import (
	"debug/elf"
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
}

func NewInputSection(ctx *Context, name string, file *ObjectFile, shndx uint32) *InputSection {
	s := &InputSection{
		File:    file,
		Shndx:   shndx,
		IsAlive: true,
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
	utils.Assert(i.Shndx < uint32(len(i.File.Sections)))
	return &i.File.InputFile.Sections[i.Shndx]
}

func (i *InputSection) Name() string {
	return GetNameFromTable(i.File.StrTable, i.Shdr().Name)
}

func (i *InputSection) WriteTo(buf []byte) {
	if i.Shdr().Type == uint32(elf.SHT_NOBITS) || i.ShSize == 0 {
		return
	}

	i.CopyContents(buf)
}

func (i *InputSection) CopyContents(buf []byte) {
	copy(buf, i.Contents)
}
