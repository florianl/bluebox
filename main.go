package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cavaliergopher/cpio"
)

var (
	output string
	arch   string
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
}

func usage() {
	cmd := filepath.Base(os.Args[0])
	fmt.Printf("%s creates a bootable initramfs, that will embed the given statically "+
		"linked executables.\n\n", cmd)
	flag.PrintDefaults()
}

// addFile adds file to the cpio archive.
func addFile(w *cpio.Writer, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if err := w.WriteHeader(&cpio.Header{
		Name: filepath.Base(file),
		Mode: cpio.FileMode(fi.Mode().Perm()),
		Size: fi.Size(),
	}); err != nil {
		return err
	}
	if _, err := io.Copy(w, f); err != nil {
		return err
	}
	return w.Flush()
}

// fail print the error to stderr and calls exit.
func fail(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	archive, err := os.OpenFile(output, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		fail(err)
	}

	w := cpio.NewWriter(archive)
	defer func() {
		if err := w.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		if err := archive.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
	}()

	for _, file := range execs {
		if err := addFile(w, file); err != nil {
			fail(err)
		}
	}

	for _, file := range readOnlys {
		if err := addFile(w, file); err != nil {
			fail(err)
		}
	}

	dir, err := os.MkdirTemp("", "bluebox-")
	if err != nil {
		fail(err)
	}
	defer func() {
		os.RemoveAll(dir)
	}()

	if err := createInit(dir, execs, args); err != nil {
		fail(err)
	}

	if err := addFile(w, filepath.Join(dir, "init")); err != nil {
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
		return fmt.Errorf("embeded executable should not be named '%s'", cmd)
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
