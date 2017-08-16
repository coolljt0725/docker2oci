package main

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

func createLayoutFile(root string) error {
	var layout v1.ImageLayout
	layout.Version = v1.ImageLayoutVersion
	contents, err := json.Marshal(layout)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(root, v1.ImageLayoutFile), contents, 0644)
}

func createLayerBlob(root string, inTar io.Reader) (v1.Descriptor, error) {
	return createBlob(root, inTar)
}

func createIndexFile(root string, index v1.Index) error {
	content, err := json.Marshal(index)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(root, "index.json"), content, 0644)
}

func createManifestFile(root string, manifest v1.Manifest) (v1.Descriptor, error) {
	content, err := json.Marshal(manifest)
	if err != nil {
		return v1.Descriptor{}, err
	}

	return createBlob(root, bytes.NewBuffer(content))
}

func createConfigFile(root string, config v1.Image) (v1.Descriptor, error) {
	content, err := json.Marshal(config)
	if err != nil {
		return v1.Descriptor{}, err
	}

	return createBlob(root, bytes.NewBuffer(content))
}

func createBlob(root string, stream io.Reader) (v1.Descriptor, error) {
	name := filepath.Join(root, "blobs", "sha256", ".tmp-blob")
	err := os.MkdirAll(filepath.Dir(name), 0700)
	if err != nil {
		return v1.Descriptor{}, err
	}

	f, err := os.Create(name)
	if err != nil {
		return v1.Descriptor{}, err
	}
	defer f.Close()

	digester := digest.SHA256.Digester()
	tee := io.TeeReader(stream, digester.Hash())
	size, err := io.Copy(f, tee)
	if err != nil {
		return v1.Descriptor{}, err
	}

	if err := f.Sync(); err != nil {
		return v1.Descriptor{}, err
	}

	if err := f.Chmod(0644); err != nil {
		return v1.Descriptor{}, err
	}

	if err := digester.Digest().Validate(); err != nil {
		return v1.Descriptor{}, err
	}

	err = os.Rename(name, filepath.Join(filepath.Dir(name), digester.Digest().Hex()))
	if err != nil {
		return v1.Descriptor{}, err
	}

	return v1.Descriptor{
		Digest: digester.Digest(),
		Size:   size,
	}, nil
}
