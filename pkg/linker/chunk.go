package linker

type Chunk struct {
	Name string
	Shdr SectionHeader
}

func NewChunk() Chunk {
	return Chunk{
		Shdr: SectionHeader{
			Addralign: 1,
		},
	}
}
