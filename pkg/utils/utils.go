package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"runtime/debug"
)

func Fatal(v any) {
	fmt.Printf("rvld:\n\t\033[0;1;31mfatal\033[0m: %v\n", v)
	debug.PrintStack()
	os.Exit(1)
}

func MustNo(err error) {
	if err != nil {
		Fatal(err.Error())
		os.Exit(1)
	}
}

func Read[T any](data []byte) (val T) {
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.LittleEndian, &val)

	MustNo(err)

	return val
}

func Assert(condition bool) {
	if !condition {
		Fatal("Assert Failed")
	}
}
