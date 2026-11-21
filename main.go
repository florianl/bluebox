package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"unicode"

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
	env       map[string]string
)

var (
	executableUsage = "Embed statically linked executable into the archive and execute it once " +
		"with the resulting init.\nArgument can be specified multiple times.\n\nFormat:\n" +
		"foo:bar\t\t\tWhen foo is executed bar will be the given argument.\n" +
		"date:+%%s\t\tWhen executed it will print the date as Unix timestamp.\n" +
		"bazinga:\"-bingo -73\"\tAdd the executable bazinga with the arguments '-bingo' and '-73'.\n" +
		"qux:\"-run 'a b'\"\t\tQuotes and backslashes keep white space as part of an argument."
	readOnlyUsage = "Just embed the given file into the archive. The file will not be executed " +
		"by the resulting init.\nArgument can be specified multiple times."
	envVarUsage = "Set environment variable.\n\nFormat:\nfoo=bar\n" +
		"Argument can be specified multiple times."
)

func init() {
	flag.StringVar(&output, "o", "initramfs.cpio", "Define the name of the output file.")
	flag.StringVar(&arch, "a", "", "Target architecture of the resulting archive. All values "+
		"that are accepted by GOARCH are possible.\nBy default the host architecture is used.")
	flag.Func("e", executableUsage, embedExec)
	flag.Func("r", readOnlyUsage, embedFile)
	flag.Func("v", envVarUsage, embedEnvVar)
	flag.BoolVar(&version, "version", false, "Print revision of this bluebox executable and return.")

	env = make(map[string]string)
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

func showVersion() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Printf("bluebox@unknown\n")
		return
	}

	var rev string

	for _, bs := range bi.Settings {
		if bs.Key == "vcs.revision" {
			rev = bs.Value
			break
		}
	}
	fmt.Printf("bluebox@%s\n", rev)
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

	for k, v := range env {
		bluebox.Setenv(k, v)
	}

	if err := writeArchive(output, bluebox); err != nil {
		fail(err)
	}
}

// writeArchive generates the archive of b and writes it to the file named output.
func writeArchive(output string, b *initramfs.Bluebox) error {
	archive, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	if err := b.Generate(archive); err != nil {
		// Do not leave a partial archive behind, but keep special files like /dev/stdout.
		if fi, statErr := archive.Stat(); statErr == nil && fi.Mode().IsRegular() {
			defer os.Remove(output)
		}
		archive.Close()
		return err
	}

	return archive.Close()
}

// embedExec parses the value of the -e flag.
//
// Examples:
// foo:bar
// foo:"-v -bar"
// foo:"-run 'Test A' -v"
// foo
func embedExec(arg string) error {
	if len(arg) == 0 {
		return nil
	}

	cmd, options := splitExec(arg)
	if initramfs.IsReserved(cmd) {
		return fmt.Errorf("embedded executable should not be named '%s'", filepath.Base(cmd))
	}

	arguments, err := splitArgs(options)
	if err != nil {
		return fmt.Errorf("invalid arguments for %s: %v", cmd, err)
	}

	execs = append(execs, cmd)
	args = append(args, arguments)
	return nil
}

// splitExec splits arg into the path of the executable and its arguments. The colon that follows
// an existing file is preferred as separator, so that paths can contain colons.
func splitExec(arg string) (string, string) {
	for i, r := range arg {
		if r != ':' {
			continue
		}
		if s, err := os.Stat(arg[:i]); err == nil && !s.IsDir() {
			return arg[:i], arg[i+1:]
		}
	}

	if _, err := os.Stat(arg); err == nil {
		return arg, ""
	}

	cmd, options, _ := strings.Cut(arg, ":")
	return cmd, options
}

// splitArgs splits s at white space into arguments. Single quotes, double quotes and backslashes
// keep white space as part of an argument.
func splitArgs(s string) ([]string, error) {
	// A single pair of double quotes around all arguments only groups them for the shell.
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' && !strings.Contains(s[1:len(s)-1], `"`) {
		s = s[1 : len(s)-1]
	}

	arguments := []string{}

	var (
		cur     strings.Builder
		quote   rune
		escaped bool
		started bool
	)

	for _, r := range s {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\' && quote != '\'':
			escaped = true
			started = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			started = true
		case unicode.IsSpace(r):
			if started {
				arguments = append(arguments, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(r)
			started = true
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("missing closing %c", quote)
	}
	if escaped {
		return nil, fmt.Errorf("trailing backslash")
	}
	if started {
		arguments = append(arguments, cur.String())
	}

	return arguments, nil
}

func embedFile(file string) error {
	if initramfs.IsReserved(file) {
		return fmt.Errorf("embedded file should not be named '%s'", filepath.Base(file))
	}
	readOnlys = append(readOnlys, file)
	return nil
}

func embedEnvVar(arg string) error {
	if len(arg) == 0 {
		return nil
	}

	split := strings.SplitN(arg, "=", 2)

	if len(split) == 1 {
		env[split[0]] = "TRUE"
	} else {
		env[split[0]] = split[1]
	}

	return nil
}
