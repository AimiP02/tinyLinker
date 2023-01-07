package linker

import (
	"bytes"
	"debug/elf"
	"rvld/pkg/utils"
)

type ObjectFile struct {
	InputFile

	SymtabSection  *SectionHeader
	SymtabShndxSec []uint32
	Sections       []*InputSection

	MergeableSections []*MergeableSection
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

	// initialize Mergeable Sections
	o.InitializeMergeableSections(ctx)

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
	o.SymtabShndxSec = utils.ReadSlice[uint32](bs, 4)
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

func (o *ObjectFile) InitializeMergeableSections(ctx *Context) {
	o.MergeableSections = make([]*MergeableSection, len(o.Sections))
	for i := 0; i < len(o.Sections); i++ {
		isec := o.Sections[i]
		if isec != nil && isec.IsAlive && isec.Shdr().Flags&uint64(elf.SHF_MERGE) != 0 {
			o.MergeableSections[i] = SplitSection(ctx, isec)
			isec.IsAlive = false
		}
	}
}

func SplitSection(ctx *Context, isec *InputSection) *MergeableSection {
	m := &MergeableSection{}
	shdr := isec.Shdr()

	m.Parent = GetMergedSectionInstance(ctx, isec.Name(), shdr.Type, shdr.Flags)
	m.P2Align = isec.P2Align

	data := isec.Contents
	offset := uint64(0)

	if shdr.Flags&uint64(elf.SHF_STRINGS) != 0 {
		for len(data) > 0 {
			// string align to the shdr.Entsize
			// Example: shdr.EntSize = 4
			// "Hel" -> "H\0\0\0 e\0\0\0 l\0\0\0 \0\0\0\0"
			end := FindNull(data, int(shdr.Entsize))
			if end == -1 {
				utils.Fatal("string is not null terminated")
			}

			sz := uint64(end) + shdr.Entsize

			substr := data[:sz]
			data = data[sz:]
			m.Strs = append(m.Strs, string(substr))
			m.FragOffsets = append(m.FragOffsets, uint32(offset))
			offset += sz
		}
	} else {
		if uint64(len(data))%shdr.Entsize != 0 {
			utils.Fatal("section size is not multiple of entsize")
		}

		for len(data) > 0 {
			substr := data[:shdr.Entsize]
			data = data[shdr.Entsize:]
			m.Strs = append(m.Strs, string(substr))
			m.FragOffsets = append(m.FragOffsets, uint32(offset))
			offset += shdr.Entsize
		}
	}

	return m
}

func FindNull(data []byte, entSize int) int {
	if entSize == 1 {
		return bytes.Index(data, []byte{0})
	}

	for i := 0; i <= len(data)-entSize; i += entSize {
		bs := data[i : i+entSize]
		if utils.AllZeros(bs) {
			return i
		}
	}

	return -1
}

func (o *ObjectFile) RegisterSectionPieces() {
	for _, m := range o.MergeableSections {
		if m == nil {
			continue
		}

		m.Fragments = make([]*SectionFragment, 0, len(m.Strs))

		for i := 0; i < len(m.Strs); i++ {
			m.Fragments = append(m.Fragments, m.Parent.Insert(m.Strs[i], uint32(m.P2Align)))
		}
	}

	for i := 1; i < len(o.InputFile.SymTable); i++ {
		sym := o.Symbols[i]
		esym := &o.InputFile.SymTable[i]

		if esym.IsAbs() || esym.IsUndef() || esym.IsCommon() {
			continue
		}

		m := o.MergeableSections[o.GetShndx(esym, i)]
		if m == nil {
			continue
		}

		frag, fragOffset := m.GetFragment(uint32(esym.Value))
		if frag == nil {
			utils.Fatal("bad symbol value")
		}
		sym.SetSectionFragment(frag)
		sym.Value = uint64(fragOffset)
	}
}
