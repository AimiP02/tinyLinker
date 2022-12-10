package linker

import (
	"bytes"
	"debug/elf"
	"fmt"
	"rvld/pkg/utils"
	"unsafe"
)

type Header64 struct {
	Ident     [16]byte /* File identification. */
	Type      uint16   /* File type. */
	Machine   uint16   /* Machine architecture. */
	Version   uint32   /* ELF format version. */
	Entry     uint64   /* Entry point. */
	Phoff     uint64   /* Program header file offset. */
	Shoff     uint64   /* Section header file offset. */
	Flags     uint32   /* Architecture-specific flags. */
	Ehsize    uint16   /* Size of ELF header in bytes. */
	Phentsize uint16   /* Size of program header entry. */
	Phnum     uint16   /* Number of program header entries. */
	Shentsize uint16   /* Size of section header entry. */
	Shnum     uint16   /* Number of section header entries. */
	Shstrndx  uint16   /* Section name strings section. */
}

type SectionHeader struct {
	Name      uint32
	Type      uint32
	Flags     uint64
	Addr      uint64
	Offset    uint64
	Size      uint64
	Link      uint32
	Info      uint32
	Addralign uint64
	Entsize   uint64
}

type Sym64 struct {
	Name  uint32 /* String table index of name. */
	Info  uint8  /* Type and binding information. */
	Other uint8  /* Reserved (not used). */
	Shndx uint16 /* Section index of symbol. */
	Value uint64 /* Symbol value. */
	Size  uint64 /* Size of associated object. */
}

type InputFile struct {
	File        *File
	Sections    []SectionHeader
	FirstGlobal int64
	SymTable    []Sym64
	SymStrTable []byte
	StrTable    []byte
}

const ELFHeaderSize = unsafe.Sizeof(Header64{})
const SectionHeaderSize = unsafe.Sizeof(SectionHeader{})
const SymbolSize = unsafe.Sizeof(Sym64{})

func NewInputFile(file *File) InputFile {
	elfFile := InputFile{File: file}

	if len(file.Contents) < int(ELFHeaderSize) {
		utils.Fatal("ELF file too small!")
	}

	if !CheckMagic(file.Contents) {
		utils.Fatal("Not an ELF file!")
	}

	elfHeader := utils.Read[elf.Header64](file.Contents)

	contents := file.Contents[elfHeader.Shoff:]

	sectionHeader := utils.Read[SectionHeader](contents)
	sectionNumber := uint64(elfHeader.Shnum)

	if sectionNumber == 0 {
		sectionNumber = uint64(sectionHeader.Size)
	}

	elfFile.Sections = []SectionHeader{sectionHeader}

	for sectionNumber > 1 {
		contents = contents[SectionHeaderSize:]
		elfFile.Sections = append(elfFile.Sections, utils.Read[SectionHeader](contents))
		sectionNumber--
	}

	shstrndx := uint64(elfHeader.Shstrndx)

	if shstrndx == uint64(elf.SHN_XINDEX) {
		shstrndx = uint64(sectionHeader.Link)
	}

	elfFile.StrTable = elfFile.GetBytesFromIndex(shstrndx)

	return elfFile
}

func (file *InputFile) GetBytesFromShdr(hdr *SectionHeader) []byte {
	start := hdr.Offset
	end := hdr.Offset + hdr.Size
	if uint64(len(file.File.Contents)) < end {
		utils.Fatal(
			fmt.Sprintf("Section header is out of range: %d", hdr.Offset),
		)
	}
	return file.File.Contents[start:end]
}

func (file *InputFile) GetBytesFromIndex(idx uint64) []byte {
	return file.GetBytesFromShdr(&file.Sections[idx])
}

func GetNameFromTable(strTable []byte, offset uint32) string {
	length := uint32(bytes.Index(strTable[offset:], []byte{0}))
	return string(strTable[offset : offset+length])
}

func (file *InputFile) FindSection(type_ uint32) *SectionHeader {
	for i := 0; i < len(file.Sections); i++ {
		shdr := &file.Sections[i]
		if shdr.Type == type_ {
			return shdr
		}
	}

	return nil
}

func (file *InputFile) FillUpSymbols(s *SectionHeader) {
	symContents := file.GetBytesFromShdr(s)
	symNumber := len(symContents) / int(SymbolSize)

	file.SymTable = make([]Sym64, 0, symNumber)

	for symNumber > 0 {
		file.SymTable = append(file.SymTable, utils.Read[Sym64](symContents))
		symContents = symContents[SymbolSize:]
		symNumber--
	}
}
