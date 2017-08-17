# docker2oci

A tool to convert images saved from `docker save` to [oci format image](https://github.com/opencontainers/image-spec).

## Installation

```
go get github.com/coolljt0725/docker2oci
```

## Build

To build `docker2oci`, the following build system dependencies are required:

* Go 1.8.x or above
* [vndr](https://github.com/LK4D4/vndr) tool

Build steps:
```
$ git clone https://github.com/coolljt0725/docker2oci $GOPATH/src/github.com/coolljt0725/docker2oci
$ cd $GOPATH/src/github.com/coolljt0725/docker2oci
$ make

```


## Example

```
$ docker save -o busybox.tar busybox
$ docker2oci -i busybox.tar busybox
```
or
```
$ docker save busybox | docker2oci busybox
```
Check the the image:
```
$ find busybox -type f
busybox/blobs/sha256/8baf43d43a34a0e6649c254b0200c2406fc40a501a852ba51a86ac3672dc0441
busybox/blobs/sha256/40a114053d955a2b80ee2cf6e13410b28b59594ceee9036b41e12c42d3e16615
busybox/blobs/sha256/8ac8bfaff55af948c796026ee867448c5b5b5d9dd3549f4006d9759b25d4a893
busybox/index.json
busybox/oci-layout
$ oci-image-tool validate busybox
oci-image-tool: reference "latest": OK
busybox: OK
Validation succeeded
$ mkdir bundle
$ oci-image-tool create --ref latest busybox bundle/
$ ls bundle/
config.json  rootfs

```

It also work if `docker save` save serveral images.
```
$ docker images busybox
REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
busybox             v2                  0f4cf98c1107        4 minutes ago       1.09MB
busybox             v1                  df8dde2cf4ad        4 minutes ago       1.09MB
busybox             latest              2b8fd9751c4c        13 months ago       1.09MB
$ docker save busybox | ./docker2oci busybox
$ oci-image-tool validate busybox
oci-image-tool: reference "latest": OK
oci-image-tool: reference "v1": OK
oci-image-tool: reference "v2": OK
busybox: OK
Validation succeeded

```
