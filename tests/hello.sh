#!/bin/bash

test_name=$(basename "$0" .sh)
path_name=out/test/$test_name

ld_path=bin/rvld

mkdir -p "$path_name"

cat <<EOF | riscv64-linux-gnu-gcc -o "$path_name"/a.o -c -xc -
#include <stdio.h>

int main() {
    printf("Hello World!\n");
    return 0;
}
EOF

$ld_path "$path_name"/a.o
