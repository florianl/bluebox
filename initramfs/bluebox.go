package initramfs

import "fmt"

type environment interface {
	fmt.Stringer
}

// logPrefix starts every line the generated executables print about themselves.
const logPrefix = "[            ]\t"

// maps to https://pkg.go.dev/syscall#Mount
type mount struct {
	source       string
	target       string
	fstype       string
	flags        int
	data         string
	targetCreate bool
	targetPerm   uint32
}

// String returns the Go statements that mount m. All values are emitted as quoted literals, so
// they can not break out of the generated source.
func (m mount) String() string {
	var code string

	if m.targetCreate {
		code = fmt.Sprintf(
			"	fmt.Println(%q)\n"+
				"	os.MkdirAll(%q, 0o%o)\n",
			fmt.Sprintf("%sos.MkdirAll(%q, 0o%o)", logPrefix, m.target, m.targetPerm),
			m.target, m.targetPerm)
	}

	return code + fmt.Sprintf(
		"	fmt.Println(%q)\n"+
			"	if err := syscall.Mount(%q, %q, %q, uintptr(%d), %q); err != nil {\n"+
			"		fmt.Println(%q, err)\n"+
			"	}\n",
		fmt.Sprintf("%ssyscall.Mount(%q, %q, %q, uintptr(%d), %q)", logPrefix,
			m.source, m.target, m.fstype, m.flags, m.data),
		m.source, m.target, m.fstype, m.flags, m.data,
		logPrefix+"ERROR:")
}

// maps to https://pkg.go.dev/syscall#Mknod
type nod struct {
	path string
	mode uint32
	dev  int
}

// String returns the Go statements that create n. All values are emitted as quoted literals, so
// they can not break out of the generated source.
func (n nod) String() string {
	return fmt.Sprintf(
		"	fmt.Println(%q)\n"+
			"	os.Remove(%q)\n"+
			"	fmt.Println(%q)\n"+
			"	if err := syscall.Mknod(%q, %d, 0x%x); err != nil {\n"+
			"		fmt.Println(%q, err)\n"+
			"	}\n",
		fmt.Sprintf("%sos.Remove(%q)", logPrefix, n.path),
		n.path,
		fmt.Sprintf("%ssyscall.Mknod(%q, 0x%x, 0x%x)", logPrefix, n.path, n.mode, n.dev),
		n.path, n.mode, n.dev,
		logPrefix+"ERROR:")
}
