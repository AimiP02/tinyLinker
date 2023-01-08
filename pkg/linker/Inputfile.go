package linker

import (
	"bytes"
	"debug/elf"
	"fmt"
	"rvld/pkg/utils"
	"strconv"
	"strings"
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

type ProgramHeader struct {
	Type     uint32
	Flags    uint32
	Offset   uint32
	VAddr    uint32
	PAddr    uint32
	FileSize uint32
	MemSize  uint32
	Align    uint64
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

type ArHeadher struct {
	Name [16]byte
	Date [12]byte
	Uid  [6]byte
	Gid  [6]byte
	Mode [8]byte
	Size [10]byte
	Fmag [2]byte
}

type InputFile struct {
	File         *File
	Sections     []SectionHeader
	FirstGlobal  int
	SymTable     []Sym64
	SymStrTable  []byte
	StrTable     []byte
	IsAlive      bool
	Symbols      []*Symbol
	LocalSymbols []Symbol
}

const IMAGE_BASE uint64 = 0x200000
const EF_RISCV_RVC uint32 = 1
const ELFHeaderSize = unsafe.Sizeof(Header64{})
const ProgramHeaderSize = unsafe.Sizeof(ProgramHeader{})
const SectionHeaderSize = unsafe.Sizeof(SectionHeader{})
const SymbolSize = unsafe.Sizeof(Sym64{})
const ArHeaderSize = unsafe.Sizeof(ArHeadher{})

// InputFile method
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

func (file *InputFile) GetEhdr() Header64 {
	return utils.Read[Header64](file.File.Contents)
}

func (file *InputFile) FillUpSymbols(s *SectionHeader) {
	symContents := file.GetBytesFromShdr(s)
	file.SymTable = utils.ReadSlice[Sym64](symContents, int(SymbolSize))
}

// ArHeader methods
func (a *ArHeadher) GetSize() int {
	size, err := strconv.Atoi(strings.TrimSpace(string(a.Size[:])))
	utils.MustNo(err)
	return size
}

func (a *ArHeadher) HasPrefix(s string) bool {
	return strings.HasPrefix(string(a.Name[:]), s)
}

func (a *ArHeadher) IsStrtab() bool {
	return a.HasPrefix("// ")
}

func (a *ArHeadher) IsSymtab() bool {
	return a.HasPrefix("/ ") || a.HasPrefix("/SYM64/ ")
}

func (a *ArHeadher) ReadName(strTab []byte) string {
	// if name is too long, it will store in string table
	if a.HasPrefix("/") {
		start, err := strconv.Atoi(strings.TrimSpace(string(a.Name[1:])))
		utils.MustNo(err)

		end := start + bytes.Index(strTab[start:], []byte("/\n"))
		return string(strTab[start:end])
	}

	// else name will store in "Name"
	nameEnd := bytes.Index(a.Name[:], []byte("/"))
	utils.Assert(nameEnd != -1)

	return string(a.Name[:nameEnd])
}

// Sym64 methods
func (s *Sym64) IsAbs() bool {
	return s.Shndx == uint16(elf.SHN_ABS)
}

func (s *Sym64) IsUndef() bool {
	return s.Shndx == uint16(elf.SHN_UNDEF)
}

func (s *Sym64) IsCommon() bool {
	return s.Shndx == uint16(elf.SHN_COMMON)
}
