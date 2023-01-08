package linker

type Chunker interface {
	GetName() string
	GetShdr() *SectionHeader
	UpdateShdr(ctx *Context)
	GetShndx() int64
	CopyBuf(ctx *Context)
}

type Chunk struct {
	Name  string
	Shdr  SectionHeader
	Shndx int64
}

func NewChunk() Chunk {
	return Chunk{
		Shdr: SectionHeader{
			Addralign: 1,
		},
	}
}

func (c *Chunk) GetName() string {
	return c.Name
}

func (c *Chunk) GetShdr() *SectionHeader {
	return &c.Shdr
}

func (c *Chunk) UpdateShdr(ctx *Context) {}

func (c *Chunk) GetShndx() int64 {
	return c.Shndx
}

func (c *Chunk) CopyBuf(ctx *Context) {

}
