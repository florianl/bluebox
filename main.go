package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/florianl/bluebox/initramfs"
)

var (
	output  string
	arch    string
	version bool
)

var (
	execs     []string
	readOnlys []string
	args      [][]string
)

var (
	executableUsage = "Embed statically linked executable into the archive and execute it once " +
		"with the resulting init.\nArgument can be specified multiple times.\n\nFormat:\n" +
		"foo:bar\t\t\tWhen foo is executed bar will be the given argument.\n" +
		"date:+%%s\t\tWhen executed it will print the date as Unix timestamp.\n" +
		"bazinga:\"-bingo -73\"\tAdd the executable bazinga with the arguments '-bingo' and '-73'."
	readOnlyUsage = "Just embed the given file into the archive. The file will not be executed " +
		"by the resulting init.\nArgument can be specified multiple times."
)

func init() {
	flag.StringVar(&output, "o", "initramfs.cpio", "Define the name of the output file.")
	flag.StringVar(&arch, "a", "", "Target architecture of the resulting archive. All values "+
		"that are accepted by GOARCH are possible.\nBy default the host architecture is used.")
	flag.Func("e", executableUsage, embedExec)
	flag.Func("r", readOnlyUsage, embedFile)
	flag.BoolVar(&version, "version", false, "Print revision of this bluebox executable and return.")
}

func usage() {
	cmd := filepath.Base(os.Args[0])
	fmt.Printf("%s creates a bootable initramfs, that will embed the given statically "+
		"linked executables.\n\n", cmd)
	flag.PrintDefaults()
}

// fail print the error to stderr and calls exit.
func fail(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if version {
		showVersion()
		return
	}

	bluebox := initramfs.New()

	for i := range execs {
		if err := bluebox.Execute(execs[i], args[i]...); err != nil {
			fail(err)
		}
	}

	for _, file := range readOnlys {
		if err := bluebox.Embed(file); err != nil {
			fail(err)
		}
	}

	if arch != "" {
		if err := bluebox.Setarch(arch); err != nil {
			fail(err)
		}
	}
	archive, err := os.OpenFile(output, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		fail(err)
	}
	defer archive.Close()
	if err := bluebox.Generate(archive); err != nil {
		fail(err)
	}
}

// Examples:
// foo:bar
// foo:"-v -bar"
// foo
func embedExec(arg string) error {
	if len(arg) == 0 {
		return nil
	}
	split := strings.SplitN(arg, ":", 2)

	cmd := split[0]
	if cmd == "init" || cmd == "bluebox-init" || cmd == "bluebox" {
		return fmt.Errorf("embedded executable should not be named '%s'", cmd)
	}
	execs = append(execs, cmd)

	if len(split) == 1 {
		args = append(args, []string{})
		return nil
	}
	options := strings.TrimPrefix(split[1], "\"")
	options = strings.TrimSuffix(options, "\"")
	arguments := strings.Split(options, " ")
	args = append(args, arguments)
	return nil
}

func embedFile(file string) error {
	readOnlys = append(readOnlys, file)
	return nil
}
