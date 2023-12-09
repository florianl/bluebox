bluebox [![Go](https://github.com/florianl/bluebox/actions/workflows/tests.yml/badge.svg?branch=main)](https://github.com/florianl/bluebox/actions/workflows/tests.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/florianl/bluebox.svg)](https://pkg.go.dev/github.com/florianl/bluebox)
=======

`bluebox` is intended to fast build a low overhead environment to be able to run tests against [Linux kernel](https://kernel.org/) APIs like [netlink](https://man7.org/linux/man-pages/man7/netlink.7.html) or [ebpf](https://man7.org/linux/man-pages/man2/bpf.2.html). It embeds given statically linked executables into the resulting archive. In a virtual environment with this archive the embedded executables will be executed in a sequential order.
`bluebox` does not provide a shell or other executables.

## Installation

```
$ go install github.com/florianl/bluebox@latest
```

## API

_Note: APIs subject to change while `bluebox` is still in an experimental phase. You can use it but we suggest you pin a version with your package manager of choice._

## Example usage

In the following example `qemu-system-x86_64` is required to start the virtual environment. For the kernel image a self compiled kernel or a prepared kernel like they are offered by [github.com/cilium/ci-kernels](https://github.com/cilium/ci-kernels) can be used. If the kernel is compiled for a different architecture, then a different version of `qemu` is required as well `bluebox` also need to know about the target architecture.

```
  # Generate a very basic initial ramdisk
$ bluebox -o my-initramfs.cpio
  # Boot a kernel in a virtual environment with the generated archive
$ qemu-system-x86_64 -m 4096 -kernel my-linux.bz -initrd my-initramfs.cpio
```

A more detailed example of how `bluebox` can be used is given in [EXAMPLE.md](https://github.com/florianl/bluebox/blob/main/EXAMPLE.md).

## Requirements

A version of Go that is [supported by upstream](https://golang.org/doc/devel/release.html#policy)

## Similar projects

- [busybox](https://www.busybox.net)
- [toybox](https://landley.net/toybox/)
- [u-root](https://github.com/u-root/u-root)
- [virtme](https://git.kernel.org/pub/scm/utils/kernel/virtme/virtme.git/)
- [virtme-ng](https://github.com/arighi/virtme-ng)
