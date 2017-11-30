package main

// +build ignore

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

//nolint: gas
func main() {
	bytePattern := regexp.MustCompile("[[][]]byte")
	const oldPath = "types/types.pb.go"
	const tmpPath = "types/types.pb.new"
	content, err := ioutil.ReadFile(oldPath)
	if err != nil {
		panic("cannot read " + oldPath)
		os.Exit(1)
	}
	lines := bytes.Split(content, []byte("\n"))
	outFile, _ := os.Create(tmpPath)
	wroteImport := false
	for _, line_bytes := range lines {
		line := string(line_bytes)
		gotPackageLine := strings.HasPrefix(line, "package ")
		writeImportTime := strings.HasPrefix(line, "import ")
		containsDescriptor := strings.Contains(line, "Descriptor")
		containsByteArray := strings.Contains(line, "[]byte")
		if containsByteArray && !containsDescriptor {
			line = string(bytePattern.ReplaceAll([]byte(line), []byte("data.Bytes")))
		}
		if writeImportTime && !wroteImport {
			wroteImport = true
			fmt.Fprintf(outFile, "import \"github.com/tendermint/go-wire/data\"\n")

		}
		if gotPackageLine {
			fmt.Fprintf(outFile, "%s\n", "//nolint: gas")
		}
		fmt.Fprintf(outFile, "%s\n", line)
	}
	outFile.Close()
	os.Remove(oldPath)
	os.Rename(tmpPath, oldPath)
	exec.Command("goimports", "-w", oldPath)
}
