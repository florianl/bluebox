## Overview

The following is a step by step walk through showcasing `bluebox` in order to test code against different Linux kernel versions.

## Required software

In this example the following software is used.

- [bluebox](https://github.com/florianl/bluebox)
- [git](https://git-scm.com/)
- [go](https://go.dev/)
- [qemu](https://www.qemu.org/)
- [docker buildx](https://docs.docker.com/reference/cli/docker/buildx/)

## Get pre-compiled Linux kernel
To get all pre-compiled [Linux kernels](https://kernel.org/) from the [cilium/ci-kernels](https://github.com/cilium/ci-kernels) repository [docker buildx](https://docs.docker.com/reference/cli/docker/buildx/) is required.

```
$ mkdir /tmp/ci-kernel
$ echo "FROM ghcr.io/cilium/ci-kernels:4.9" | docker buildx build --quiet --pull --output="/tmp/ci-kernel" -
```

## Prepare testing code
This walk through will test the Linux [netlink](https://man7.org/linux/man-pages/man7/netlink.7.html) API using tests from the [Go](https://go.dev/) package [mdlayher/netlink](https://github.com/mdlayher/netlink). In a first step get the code:
```
$ cd /tmp
$ git clone --depth 1 https://github.com/mdlayher/netlink.git
```
Then build a statically linked executable from the included tests in this repository.
```
$ cd /tmp/netlink
$ go test -ldflags='-extldflags=-static' -trimpath -tags 'osusergo netgo static_build linux' -c
```

## Create the initramfs.cpio with `bluebox`
Create an archive that can be used as [initial ramdisk](https://en.wikipedia.org/wiki/Initial_ramdisk) and embedd the statically linked executable.
```
$ cd /tmp
$ bluebox -e /tmp/netlink/netlink.test:"-test.v"
```
As argument `-test.v` is passed to `netlink.test` once this binary is executed.

## Run the tests in a virtual machine
The shown [`qemu-system-x86_64`](https://www.qemu.org/) command will start the pre-compiled Linux kernel from [cilium/ci-kernels](https://github.com/cilium/ci-kernels) and use the archive that was genereated by `bluebox` as initial ramdisk.
```
$ qemu-system-x86_64 -nographic -append "console=ttyS0" -m 4G -kernel /tmp/ci-kernel/boot/vmlinuz -initrd /tmp/initramfs.cpio

[...]

[            ]	./netlink.test exited, exit status 1
[            ] stdout
=== RUN   Test_nlmsgAlign
=== RUN   Test_nlmsgAlign/0
=== RUN   Test_nlmsgAlign/1
=== RUN   Test_nlmsgAlign/2
=== RUN   Test_nlmsgAlign/3
=== RUN   Test_nlmsgAlign/4
=== RUN   Test_nlmsgAlign/5
=== RUN   Test_nlmsgAlign/6
=== RUN   Test_nlmsgAlign/7
=== RUN   Test_nlmsgAlign/8
--- PASS: Test_nlmsgAlign (0.00s)
    --- PASS: Test_nlmsgAlign/0 (0.00s)
    --- PASS: Test_nlmsgAlign/1 (0.00s)
    --- PASS: Test_nlmsgAlign/2 (0.00s)
    --- PASS: Test_nlmsgAlign/3 (0.00s)

[...]

=== RUN   TestIntegrationConnSetBuffersSyscallConn/privileged
    conn_linux_integration_test.go:897: $ ip [tuntap add nlprobe0 mode tun]
    conn_linux_integration_test.go:897: failed to start command "ip": exec: "ip": executable file not found in $PATH
=== RUN   ExampleAttributeDecoder_decode
--- PASS: ExampleAttributeDecoder_decode (0.00s)
=== RUN   ExampleAttributeEncoder_encode
--- PASS: ExampleAttributeEncoder_encode (0.00s)
FAIL
```

`bluebox` creates a minimal archive that can be used as initial ramdisk. Additional executables like [`ip`](https://man7.org/linux/man-pages/man8/ip.8.html) are not included. So the test `TestIntegrationConnSetBuffersSyscallConn` is expected to fail. Tests that interact with the [netlink](https://man7.org/linux/man-pages/man7/netlink.7.html) API of the [Linux kernel](https://kernel.org/) without such an external dependency pass.

## CI/CD

The [Github Action](https://docs.github.com/en/actions) workflow defined by [example.yml](https://github.com/florianl/bluebox/blob/main/.github/workflows/example.yml) in this repository showcases the use of `bluebox` in a CI/CD setup.
