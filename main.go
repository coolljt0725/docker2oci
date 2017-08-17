package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	specs "github.com/opencontainers/image-spec/specs-go"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/urfave/cli"
)

var gitCommit = ""

const (
	manifestFileName           = "manifest.json"
	legacyLayerFileName        = "layer.tar"
	legacyRepositoriesFileName = "repositories"
)

var appHelpTemplate = `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText}}{{end}} {{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .VisibleFlags}}

OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}
`

func main() {
	cli.AppHelpTemplate = fmt.Sprintf("%s", appHelpTemplate)
	app := cli.NewApp()
	app.Name = "docker2oci"
	app.Usage = "convert docker image from docker save to oci format image"
	app.UsageText = "docker2oci [OPTIONS] DIRECTORY"
	app.Version = fmt.Sprintf("commit: %s spec version: %s", gitCommit, specs.Version)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "i,input",
			Value: "",
			Usage: "Read image from tar archive `FILE`, instead of STDIN",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("Error: destination 'DIRECTORY' of oci image is requred, see 'docker2oci --help'")
		}
		inputfile := c.String("input")

		var (
			input io.Reader = os.Stdin
		)

		if inputfile != "" {
			infile, err := os.Open(inputfile)
			if err != nil {
				return err
			}
			defer infile.Close()
			input = infile
		}
		dir := c.Args().Get(0)
		if _, err := os.Stat(dir); err == nil {
			return fmt.Errorf("Destination %q exist", dir)
		} else if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0700)
			if err != nil {
				return err
			}
		} else {
			return err
		}
		return doConvert(input, dir)

	}
	app.Run(os.Args)
}

// TODO: need a refactor to split the big function to several
// small functions
func doConvert(in io.Reader, out string) (retErr error) {
	tmpDir, err := ioutil.TempDir("", "docker2oci-docker-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := unpack(tmpDir, in); err != nil {
		return err
	}

	manifestPath := filepath.Join(tmpDir, manifestFileName)
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	var manifests []manifestItem
	if err := json.NewDecoder(manifestFile).Decode(&manifests); err != nil {
		return err
	}

	if err := createLayoutFile(out); err != nil {
		return err
	}

	var index v1.Index
	index.SchemaVersion = 2

	for _, m := range manifests {
		var manifest v1.Manifest

		manifest.SchemaVersion = 2

		configPath := filepath.Join(tmpDir, m.Config)
		config, err := ioutil.ReadFile(configPath)
		if err != nil {
			return err
		}
		img, err := NewFromJSON(config)
		if err != nil {
			return err
		}
		// TODO: use v1.Image to read config from file directly
		// TODO: move this to a function
		ociConfig := v1.Image{
			Created:      &img.Created,
			Author:       img.Author,
			Architecture: img.Architecture,
			OS:           img.OS,
			Config: v1.ImageConfig{
				User:         img.ContainerConfig.User,
				ExposedPorts: img.ContainerConfig.ExposedPorts,
				Env:          img.ContainerConfig.Env,
				Entrypoint:   []string(img.ContainerConfig.Entrypoint),
				Cmd:          []string(img.ContainerConfig.Cmd),
				Volumes:      img.ContainerConfig.Volumes,
				WorkingDir:   img.ContainerConfig.WorkingDir,
				Labels:       img.ContainerConfig.Labels,
				StopSignal:   img.ContainerConfig.StopSignal,
			},
			RootFS: v1.RootFS{
				Type:    img.RootFS.Type,
				DiffIDs: img.RootFS.DiffIDs,
			},
			History: img.History,
		}
		des, err := createConfigFile(out, ociConfig)
		if err != nil {
			return err
		}
		des.MediaType = v1.MediaTypeImageConfig
		manifest.Config = des
		for i, _ := range img.RootFS.DiffIDs {
			layerPath := filepath.Join(tmpDir, m.Layers[i])
			f, err := os.Open(layerPath)
			if err != nil {
				return err
			}
			defer f.Close()
			des, err := createLayerBlob(out, f)
			if err != nil {
				return err
			}
			// TODO: detect the tar format, so we know the mediaType
			des.MediaType = v1.MediaTypeImageLayer
			manifest.Layers = append(manifest.Layers, des)
		}
		des, err = createManifestFile(out, manifest)
		if err != nil {
			return err
		}
		des.MediaType = v1.MediaTypeImageManifest
		des.Platform = &v1.Platform{
			Architecture: ociConfig.Architecture,
			OS:           ociConfig.OS,
		}
		des.Annotations = make(map[string]string)
		// FIXME: a image may have multiple tags
		// TODO: validate the tag
		for _, tag := range m.RepoTags {
			strs := strings.Split(tag, ":")
			if len(strs) != 2 {
				continue
			}
			des.Annotations["org.opencontainers.image.ref.name"] = strs[1]
		}
		index.Manifests = append(index.Manifests, des)
	}
	err = createIndexFile(out, index)
	if err != nil {
		return err
	}

	return nil
}
