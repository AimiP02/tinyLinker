package linker

import (
	"debug/elf"
	"rvld/pkg/utils"
)

type MachineType = uint8

const (
	MachineTypeNone    MachineType = iota
	MachineTypeRISCV64 MachineType = iota
)

func GetMachineTypeFromContext(contents []byte) MachineType {
	ft := GetFileType(contents)

	switch ft {
	case FileTypeObject:
		machine := elf.Machine(utils.Read[uint16](contents[18:]))
		if machine == elf.EM_RISCV {
			class := elf.Class(contents[4])
			switch class {
			case elf.ELFCLASS64:
				return MachineTypeRISCV64
			}
		}
	}

	return MachineTypeNone
}

type MachineTypeStringer struct {
	MachineType
}

func (m MachineTypeStringer) String() string {
	switch m.MachineType {
	case MachineTypeRISCV64:
		return "riscv64"
	}

	utils.Assert(m.MachineType == MachineTypeNone)
	return ""
}
