package linker

import "rvld/pkg/utils"

type Symbol struct {
	File            *ObjectFile
	InputSection    *InputSection
	SectionFragment *SectionFragment
	Name            string
	Value           uint64
	SymIdx          int32
}

func NewSymbol(name string) *Symbol {
	s := &Symbol{
		Name: name,
	}

	return s
}

func (s *Symbol) SetInputSection(isec *InputSection) {
	s.InputSection = isec
	s.SectionFragment = nil
}

func (s *Symbol) SetSectionFragment(frag *SectionFragment) {
	s.InputSection = nil
	s.SectionFragment = frag
}

func GetSymbolByName(ctx *Context, name string) *Symbol {
	if sym, ok := ctx.SymbolMap[name]; ok {
		return sym
	}
	ctx.SymbolMap[name] = NewSymbol(name)
	return ctx.SymbolMap[name]
}

func (s *Symbol) ELFSym() *Sym64 {
	utils.Assert(s.SymIdx < int32(len(s.File.SymTable)))
	return &s.File.SymTable[s.SymIdx]
}

func (s *Symbol) Clear() {
	s.File = nil
	s.InputSection = nil
	s.SymIdx = -1
}
