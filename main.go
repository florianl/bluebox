package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cavaliergopher/cpio"
)

var (
	output string
	dir    string
	arch   string
)

func init() {
	flag.StringVar(&output, "o", "initramfs.cpio", "Define the name of the output file.")
	flag.StringVar(&dir, "d", "", "Directory with statically linked executables that will be "+
		"included in the archive. Subdirectories within this directory will be skipped.")
	flag.StringVar(&arch, "a", "", "Target architecture of the resulting archive. All values "+
		"that are accepted by GOARCH are possible. By default the host architecture is used.")
}

func usage() {
	cmd := filepath.Base(os.Args[0])
	fmt.Printf("%s creates a bootable initramfs, that will embed the given statically linked executables.\n\n", cmd)
	fmt.Printf("Usage:\n\t%s [Options] [Executable]...\n\n", cmd)
	fmt.Printf("Executable:\n\tList of statically linked executables with their respective arguments.\n")
	fmt.Printf("\tExamples:\n")
	fmt.Printf("\t\tfoo:bar\n\t\tWhen foo is executed bar will be the given argument.\n")
	fmt.Printf("\t\tdate:+%%s\n\t\tWhen executed it will print the date as Unix timestamp.\n")
	fmt.Printf("\t\tbazinga:\"-bingo -73\"\n\t\tAdd the executable bazinga with the arguments '-bingo' and '-73'.\n")
	fmt.Printf("Options:\n")
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

func main() {
	var execs []string
	var args [][]string

	flag.Usage = usage
	flag.Parse()
	if dir != "" {
		var err error
		execs, err = getExecs(dir)
		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < len(execs); i++ {
			args = append(args, []string{})
		}
	}

	if len(flag.Args()) != 0 {
		var err error
		execs, args, err = parseAdditionalExecs(flag.Args())
		if err != nil {
			log.Fatal(err)
		}
	}

	archive, err := os.OpenFile(output, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	w := cpio.NewWriter(archive)
	defer func() {
		if err := w.Close(); err != nil {
			log.Fatal(err)
		}
		if err := archive.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	for _, exec := range execs {
		if err := addFile(w, exec); err != nil {
			log.Fatal(err)
		}
	}

	dir, err := createInit(execs, args)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		os.RemoveAll(dir)
	}()
	if err := addFile(w, filepath.Join(dir, "init")); err != nil {
		log.Fatal(err)
	}
}

func getExecs(dir string) ([]string, error) {
	var execs []string
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry,
		err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.IsDir() {
			if path == dir {
				return nil
			}
			// Don't parse subdirectories.
			return fs.SkipDir
		}
		info, _ := d.Info()
		if !info.Mode().IsRegular() {
			return nil
		}
		perm := info.Mode().Perm()
		if perm&0o111 == 0 {
			// path is not an executable
			return nil
		}
		execs = append(execs, path)
		return nil
	}); err != nil {
		return []string{}, err
	}
	return execs, nil
}

// parseAdditionalExecs handles additional arguments that can be passed and
// hold additional executables with their respective arguments separated by colon.
//
// Examples:
// foo:bar
// foo:"-v -bar"
// foo
func parseAdditionalExecs(cmdLine []string) ([]string, [][]string, error) {
	var execs []string
	var args [][]string
	for _, v := range cmdLine {
		split := strings.SplitN(v, ":", 2)
		execs = append(execs, split[0])
		if len(split) == 1 {
			args = append(args, []string{})
			continue
		}
		options := strings.TrimPrefix(split[1], "\"")
		options = strings.TrimSuffix(options, "\"")
		arguments := strings.Split(options, " ")
		args = append(args, arguments)
	}
	return execs, args, nil
}
