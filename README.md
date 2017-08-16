# docker2oci
A tool to convert images saved from `docker save` to oci format image
Example:

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
