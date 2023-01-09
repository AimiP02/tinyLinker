package linker

import "math"

type SectionFragment struct {
	OutputSection *MergedSection
	Offset        uint32
	P2Align       uint32
	IsAlive       bool
}

func NewSecitonFragment(m *MergedSection) *SectionFragment {
	return &SectionFragment{
		OutputSection: m,
		Offset:        math.MaxUint32,
	}
}

func (s *SectionFragment) GetAddr() uint64 {
	return s.OutputSection.Shdr.Addr + uint64(s.Offset)
}
