package linker

import "rvld/pkg/utils"

func ReadInputFiles(ctx *Context, remaining []string) {
	for _, arg := range remaining {
		// normal object file
		var ok bool

		if arg, ok = utils.RemovePrefix(arg, "-l"); ok {
			ReadFile(ctx, FindLibrary(ctx, arg))
		} else {
			ReadFile(ctx, MustNewFile(arg))
		}
	}
}

func ReadFile(ctx *Context, file *File) {
	ft := GetFileType(file.Contents)

	switch ft {
	case FileTypeObject:
		// Todo ...
		ctx.Objs = append(ctx.Objs, CreateObjectFile(ctx, file, false))
	case FileTypeArchive:
		for _, child := range ReadArchiveMembers(file) {
			utils.Assert(GetFileType(child.Contents) == FileTypeObject)
			ctx.Objs = append(ctx.Objs, CreateObjectFile(ctx, child, true))
		}
	default:
		utils.Fatal("unknown file type")
	}
}

func CreateObjectFile(ctx *Context, file *File, inLib bool) *ObjectFile {
	mt := GetMachineTypeFromContext(file.Contents)
	if mt != ctx.Args.Emulation {
		utils.Fatal("incompatible file type")
	}

	obj := NewObjectFile(file, !inLib)
	obj.Parse(ctx)

	return obj
}
