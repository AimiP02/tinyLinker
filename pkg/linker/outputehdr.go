package linker

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"rvld/pkg/utils"
)

type OutputEhdr struct {
	Chunk
}

func NewOutputEhdr() *OutputEhdr {
	return &OutputEhdr{
		Chunk{
			Shdr: SectionHeader{
				Flags:     uint64(elf.SHF_ALLOC),
				Size:      uint64(ELFHeaderSize),
				Addralign: 8,
			},
		},
	}
}

func (o *OutputEhdr) CopyBuf(ctx *Context) {
	ehdr := &Header64{}
	WriteMagic(ehdr.Ident[:])
	ehdr.Ident[elf.EI_CLASS] = uint8(elf.ELFCLASS64)
	ehdr.Ident[elf.EI_DATA] = uint8(elf.ELFDATA2LSB)
	ehdr.Ident[elf.EI_VERSION] = uint8(elf.EV_CURRENT)
	ehdr.Ident[elf.EI_OSABI] = 0
	ehdr.Ident[elf.EI_ABIVERSION] = 0

	ehdr.Type = uint16(elf.ET_EXEC) // Executable file
	ehdr.Machine = uint16(elf.EM_RISCV)
	ehdr.Version = uint32(elf.EV_CURRENT)
	ehdr.Entry = GetEntryAddress(ctx)
	ehdr.Phoff = ctx.Phdr.Shdr.Offset
	ehdr.Shoff = ctx.Shdr.Shdr.Offset
	ehdr.Ehsize = uint16(ELFHeaderSize)
	ehdr.Phentsize = uint16(ProgramHeaderSize)
	ehdr.Phnum = uint16(ctx.Phdr.Shdr.Size) / uint16(ProgramHeaderSize)
	ehdr.Shentsize = uint16(SectionHeaderSize)
	ehdr.Shnum = uint16(ctx.Shdr.Shdr.Size) / uint16(SectionHeaderSize)

	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.LittleEndian, ehdr)
	utils.MustNo(err)
	copy(ctx.Buf[o.Shdr.Offset:], buf.Bytes())
}

func GetEntryAddress(ctx *Context) uint64 {
	for _, osec := range ctx.OutputSections {
		if osec.Name == ".text" {
			return osec.Shdr.Addr
		}
	}
	return 0
}

func GetFlags(ctx *Context) uint32 {
	utils.Assert(len(ctx.Objs) > 0)
	flags := ctx.Objs[0].GetEhdr().Flags

	for _, obj := range ctx.Objs[1:] {
		if obj.GetEhdr().Flags&EF_RISCV_RVC != 0 {
			flags |= EF_RISCV_RVC
			break
		}
	}
	return flags
}
