package linker

import "rvld/pkg/utils"

type InputSection struct {
	File     *ObjectFile
	Contents []byte
	Shndx    uint32
}

func NewInputSection(file *ObjectFile, shndx uint32) *InputSection {
	s := &InputSection{
		File:  file,
		Shndx: shndx,
	}

	shdr := s.Shdr()
	s.Contents = file.File.Contents[shdr.Offset : shdr.Offset+shdr.Size]

	return s
}

func (i *InputSection) Shdr() *SectionHeader {
	utils.Assert(i.Shndx < uint32(len(i.File.Sections)))
	return &i.File.InputFile.Sections[i.Shndx]
}

func (i *InputSection) Name() string {
	return GetNameFromTable(i.File.StrTable, i.Shdr().Name)
}
