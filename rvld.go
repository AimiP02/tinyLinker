package main

import (
	"os"
	"rvld/pkg/linker"
	"rvld/pkg/utils"
)

func main() {
	if len(os.Args) < 2 {
		utils.Fatal("wrong anser")
		os.Exit(1)
	}

	file := linker.MustNewFile(os.Args[1])
	objFile := linker.NewObjectFile(file)

	objFile.Parse()

	utils.Assert(len(objFile.Sections) == 12)
	utils.Assert(objFile.FirstGlobal == 12)
	utils.Assert(len(objFile.SymTable) == 14)

	for _, sym := range objFile.SymTable {
		println(linker.GetNameFromTable(objFile.SymStrTable, sym.Name))
	}
}
