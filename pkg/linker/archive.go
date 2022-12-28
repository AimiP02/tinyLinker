package linker

import "rvld/pkg/utils"

func ReadArchiveMembers(file *File) []*File {
	utils.Assert(GetFileType(file.Contents) == FileTypeArchive)

	// skip 8 bytes "!<arch>\n"
	pos := 8

	var strTab []byte
	var files []*File
	// Length of section which cannot divided by 2 will fill "\n" to align 2 bytes
	for len(file.Contents)-pos > 1 {
		if pos%2 == 1 {
			pos++
		}

		hdr := utils.Read[ArHeadher](file.Contents[pos:])
		dataStart := pos + int(ArHeaderSize)
		pos = dataStart + hdr.GetSize()
		dataEnd := pos
		contents := file.Contents[dataStart:dataEnd]

		if hdr.IsSymtab() {
			continue
		} else if hdr.IsStrtab() {
			strTab = contents
			continue
		}

		files = append(files, &File{
			Name:     hdr.ReadName(strTab),
			Contents: contents,
			Parent:   file,
		})
	}

	return files
}
