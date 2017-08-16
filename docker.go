// TODO: import github.com/moby/moby/image ?
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

// V1Image stores the V1 image configuration.
type V1Image struct {
	// Created timestamp when image was created
	Created time.Time `json:"created"`
	// ContainerConfig is the configuration of the container that is committed into the image
	ContainerConfig Config `json:"container_config,omitempty"`
	// Author of the image
	Author string `json:"author,omitempty"`
	// Architecture is the hardware that the image is build and runs on
	Architecture string `json:"architecture,omitempty"`
	// OS is the operating system used to build and run the image
	OS string `json:"os,omitempty"`
	// Size is the total size of the image including all layers it is composed of
	Size int64 `json:",omitempty"`
}

// Image stores the image configuration
type Image struct {
	V1Image
	RootFS  *RootFS      `json:"rootfs,omitempty"`
	History []v1.History `json:"history,omitempty"`
}

// Config contains the configuration data about a container.
// It should hold only portable information about the container.
// Here, "portable" means "independent from the host we are running on".
// Non-portable information *should* appear in HostConfig.
// All fields added to this struct must be marked `omitempty` to keep getting
// predictable hashes from the old `v1Compatibility` configuration.
type Config struct {
	User         string              // User that will run the command(s) inside the container
	ExposedPorts map[string]struct{} `json:",omitempty"` // List of exposed ports
	Env          []string            // List of environment variable to set in the container
	Cmd          StrSlice            // Command to run when starting the container
	Volumes      map[string]struct{} // List of volumes (mounts) used for the container
	WorkingDir   string              // Current directory (PWD) in the command will be launched
	Entrypoint   StrSlice            // Entrypoint to run when starting the container
	Labels       map[string]string   // List of labels set to this container
	StopSignal   string              `json:",omitempty"` // Signal to stop a container
}

type RootFS struct {
	Type    string          `json:"type"`
	DiffIDs []digest.Digest `json:"diff_ids,omitempty"`
}

// NewFromJSON creates an Image configuration from json.
func NewFromJSON(src []byte) (*Image, error) {
	img := &Image{}

	if err := json.Unmarshal(src, img); err != nil {
		return nil, err
	}
	if img.RootFS == nil {
		return nil, fmt.Errorf("Invalid image JSON, no RootFS key.")
	}

	return img, nil
}

// StrSlice represents a string or an array of strings.
// We need to override the json decoder to accept both options.
type StrSlice []string

// UnmarshalJSON decodes the byte slice whether it's a string or an array of
// strings. This method is needed to implement json.Unmarshaler.
func (e *StrSlice) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		// With no input, we preserve the existing value by returning nil and
		// leaving the target alone. This allows defining default values for
		// the type.
		return nil
	}

	p := make([]string, 0, 1)
	if err := json.Unmarshal(b, &p); err != nil {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		p = append(p, s)
	}

	*e = p
	return nil
}

type manifestItem struct {
	Config   string
	RepoTags []string
	Layers   []string
	Parent   digest.Digest `json:",omitempty"`
}
