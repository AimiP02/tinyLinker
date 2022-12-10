package linker

import "debug/elf"

type ObjectFile struct {
	InputFile

	SymtabSection *SectionHeader
}

func NewObjectFile(file *File) *ObjectFile {
	o := &ObjectFile{InputFile: NewInputFile(file)}
	return o
}

func (o *ObjectFile) Parse() {
	o.SymtabSection = o.FindSection(uint32(elf.SHT_SYMTAB))
	if o.SymtabSection != nil {
		o.FirstGlobal = int64(o.SymtabSection.Info)
		o.FillUpSymbols(o.SymtabSection)
		o.SymStrTable = o.GetBytesFromIndex(uint64(o.SymtabSection.Link))
	}

}
