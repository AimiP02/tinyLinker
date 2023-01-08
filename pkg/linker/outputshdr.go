package linker

import "rvld/pkg/utils"

type OutputShdr struct {
	Chunk
}

func NewOutputShdr() *OutputShdr {
	o := &OutputShdr{
		Chunk: NewChunk(),
	}
	o.Shdr.Addralign = 8
	return o
}

func (o *OutputShdr) UpdateShdr(ctx *Context) {
	n := uint64(0)
	for _, chunk := range ctx.Chunks {
		if chunk.GetShndx() > 0 {
			n = uint64(chunk.GetShndx())
		}
	}

	o.Shdr.Size = (n + 1) * uint64(SectionHeaderSize)
}

func (o *OutputShdr) CopyBuf(ctx *Context) {
	base := ctx.Buf[o.Shdr.Offset:]
	utils.Write[SectionHeader](base, SectionHeader{})

	for _, chunk := range ctx.Chunks {
		if chunk.GetShndx() > 0 {
			utils.Write[SectionHeader](base[chunk.GetShndx()*int64(SectionHeaderSize):], *chunk.GetShdr())
		}
	}
}
