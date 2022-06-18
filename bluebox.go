package main

import "fmt"

type Bluebox struct {
	Executables []string
	Arguments   [][]string
	Environment []Environment
}

type Environment interface {
	fmt.Stringer
}

// maps to https://pkg.go.dev/syscall#Mount
type Mount struct {
	source       string
	target       string
	fstype       string
	flags        int
	data         string
	targetCreate bool
	targetPerm   uint32
}

func (m Mount) String() string {
	if m.targetCreate {
		return fmt.Sprintf(
			"	fmt.Println(\"[            ]\tos.MkdirAll(\\\"%s\\\", 0o%o)\")\n"+
				"	os.MkdirAll(%q, 0o%o)\n"+
				"	fmt.Println(\"[            ]\tsyscall.Mount(\\\"%s\\\", \\\"%s\\\", \\\"%s\\\", uintptr(%d), \\\"%s\\\")\")\n"+
				"	if err := syscall.Mount(%q, %q, %q, uintptr(%d), %q); err != nil {\n"+
				"		fmt.Println(\"[            ]\tERROR:\", err)\n"+
				"	}\n",
			m.target, m.targetPerm,
			m.target, m.targetPerm,
			m.source, m.target, m.fstype, m.flags, m.data,
			m.source, m.target, m.fstype, m.flags, m.data)
	}
	return fmt.Sprintf(
		"	fmt.Println(\"[            ]\tsyscall.Mount(\\\"%s\\\", \\\"%s\\\", \\\"%s\\\", uintptr(%d), \\\"%s\\\")\")\n"+
			"	if err := syscall.Mount(%q, %q, %q, uintptr(%d), %q); err != nil {\n"+
			"		fmt.Println(\"[            ]\tERROR:\", err)\n"+
			"	}\n",
		m.source, m.target, m.fstype, m.flags, m.data,
		m.source, m.target, m.fstype, m.flags, m.data)
}

// maps to https://pkg.go.dev/syscall#Mknod
type Nod struct {
	path string
	mode uint32
	dev  int
}

func (n Nod) String() string {
	return fmt.Sprintf(
		"	fmt.Println(\"[            ]\tos.Remove(\\\"%s\\\")\")\n"+
			"	os.Remove(%q)\n"+
			"	fmt.Println(\"[            ]\tsyscall.Mknod(\\\"%s\\\", 0x%x, 0x%x)\")\n"+
			"	if err := syscall.Mknod(%q, %d, 0x%x); err != nil {\n"+
			"		fmt.Println(\"[            ]\tERROR:\", err)\n"+
			"	}\n",
		n.path,
		n.path,
		n.path, n.mode, n.dev,
		n.path, n.mode, n.dev,
	)
}
