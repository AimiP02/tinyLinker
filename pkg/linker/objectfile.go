package linker

import (
	"debug/elf"
	"rvld/pkg/utils"
)

type ObjectFile struct {
	InputFile

	SymtabSection  *SectionHeader
	SymtabShndxSec []uint32
	Sections       []*InputSection
}

func NewObjectFile(file *File, isAlive bool) *ObjectFile {
	o := &ObjectFile{InputFile: NewInputFile(file)}
	o.IsAlive = isAlive
	return o
}

func (o *ObjectFile) Parse(ctx *Context) {
	o.SymtabSection = o.FindSection(uint32(elf.SHT_SYMTAB))
	if o.SymtabSection != nil {
		o.FirstGlobal = int(o.SymtabSection.Info)
		o.FillUpSymbols(o.SymtabSection)
		o.SymStrTable = o.GetBytesFromIndex(uint64(o.SymtabSection.Link))
	}

	// initialize Sections
	o.InitializeSections()
	o.InitializeSymbols(ctx)

}

func (o *ObjectFile) InitializeSections() {
	o.Sections = make([]*InputSection, len(o.InputFile.Sections))

	for i := 0; i < len(o.InputFile.Sections); i++ {
		shdr := &o.InputFile.Sections[i]
		switch elf.SectionType(shdr.Type) {
		case elf.SHT_GROUP, elf.SHT_SYMTAB, elf.SHT_STRTAB, elf.SHT_REL, elf.SHT_RELA, elf.SHT_NULL:
			break
		case elf.SHT_SYMTAB_SHNDX:
			o.FillUpSymtabShndxSec(shdr)
		default:
			o.Sections[i] = NewInputSection(o, uint32(i))
		}
	}
}

func (o *ObjectFile) FillUpSymtabShndxSec(s *SectionHeader) {
	bs := o.GetBytesFromShdr(s)
	nums := len(bs) / 4
	for nums > 0 {
		o.SymtabShndxSec = append(o.SymtabShndxSec, utils.Read[uint32](bs))
		bs = bs[4:]
		nums--
	}
}

func (o *ObjectFile) InitializeSymbols(ctx *Context) {
	if o.SymtabSection == nil {
		return
	}

	o.LocalSymbols = make([]Symbol, o.FirstGlobal)
	for i := 0; i < int(o.FirstGlobal); i++ {
		o.LocalSymbols[i] = *NewSymbol("")
	}

	// Skip the first symbol
	o.LocalSymbols[0].File = o

	for i := 1; i < int(o.FirstGlobal); i++ {
		esym := &o.SymTable[i]
		sym := &o.LocalSymbols[i]
		sym.Name = GetNameFromTable(o.SymStrTable, esym.Name)
		sym.File = o
		sym.Value = esym.Value
		sym.SymIdx = int32(i)

		if !esym.IsAbs() {
			sym.SetInputSection(o.Sections[o.GetShndx(esym, i)])
		}
	}

	o.Symbols = make([]*Symbol, len(o.InputFile.SymTable))
	for i := 0; i < len(o.LocalSymbols); i++ {
		o.Symbols[i] = &o.LocalSymbols[i]
	}

	for i := len(o.LocalSymbols); i < len(o.InputFile.SymTable); i++ {
		esym := &o.InputFile.SymTable[i]
		name := GetNameFromTable(o.InputFile.SymStrTable, esym.Name)
		o.Symbols[i] = GetSymbolByName(ctx, name)
	}
}

func (o *ObjectFile) GetShndx(esym *Sym64, idx int) int64 {
	utils.Assert(idx >= 0 && idx < len(o.SymTable))
	if esym.Shndx == uint16(elf.SHN_XINDEX) {
		return int64(o.SymtabShndxSec[idx])
	}
	return int64(esym.Shndx)
}

func (o *ObjectFile) ResolveSymbols() {
	for i := o.FirstGlobal; i < len(o.InputFile.SymTable); i++ {
		sym := o.Symbols[i]
		esym := &o.InputFile.SymTable[i]

		if esym.IsUndef() {
			continue
		}

		var isec *InputSection
		if !esym.IsAbs() {
			isec = o.GetSecion(esym, i)
			if isec == nil {
				continue
			}
		}

		if sym.File == nil {
			sym.File = o
			sym.SetInputSection(isec)
			sym.Value = esym.Value
			sym.SymIdx = int32(i)
		}
	}
}

func (o *ObjectFile) GetSecion(esym *Sym64, idx int) *InputSection {
	return o.Sections[o.GetShndx(esym, idx)]
}

func (o *ObjectFile) MarkLiveObjects(ctx *Context, feeder func(*ObjectFile)) {
	utils.Assert(o.IsAlive)

	for i := o.FirstGlobal; i < len(o.InputFile.SymTable); i++ {
		sym := o.Symbols[i]
		esym := &o.InputFile.SymTable[i]

		if sym.File == nil {
			continue
		}

		if esym.IsUndef() && !sym.File.IsAlive {
			sym.File.IsAlive = true
			feeder(sym.File)
		}
	}
}

func (o *ObjectFile) ClearSymbols() {
	for _, sym := range o.Symbols[o.FirstGlobal:] {
		if sym.File == o {
			sym.Clear()
		}
	}
}
