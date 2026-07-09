// SPDX:Apache-2.0
package cmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	imgoin "github.com/groboclown/imgoin/pkg"
)

type imageOptions struct {
	image string

	size         uint64
	annotations  map[string]string
	architecture string

	// OS specifies the operating system, for example `linux` or `windows`.
	os string

	// OSVersion is an optional field specifying the operating system
	// version, for example on Windows `10.0.14393.1066`.
	osVersion string

	// OSFeatures is an optional field specifying an array of strings,
	// each listing a required OS feature (for example on Windows `win32k`).
	osFeatures []string

	// Variant is an optional field specifying a variant of the CPU, for
	// example `v7` to specify ARMv7 when architecture is `arm`.
	variant string

	// For getting the information
	conn            imgoin.RepositoryConnection
	includeContents bool

	// Targets can also include different tags.
	tags []string
}

// parseImageOptions reads in the options for the image, starting with the given index.
// It reads in the 'KEY=VALUE' argument formats.
func parseImageOptions(args []string, startIndex int) (*imageOptions, int, error) {
	if startIndex >= len(args) {
		return nil, startIndex, fmt.Errorf("image argument requires the image options")
	}
	if args[startIndex] != "" && args[startIndex][0] == '@' {
		// Read the options from the file instead.
		fileArgs, err := splitFileArgs(args[startIndex][1:])
		startIndex += 1
		if err != nil {
			return nil, startIndex, err
		}
		ret, _, err := parseImageOptions(fileArgs, 0)
		return ret, startIndex, err
	}

	ret := &imageOptions{}
	errs := make([]error, 0)
	for startIndex < len(args) {
		arg := args[startIndex]
		if len(arg) > 0 && arg[0] == '-' {
			// Start of a new argument.
			break
		}
		startIndex += 1
		if len(arg) == 0 {
			// Skip empty arguments.
			continue
		}

		pos := strings.IndexRune(arg, '=')
		if pos <= 0 {
			errs = append(errs, fmt.Errorf("invalid image argument (%s); must be in the format 'KEY=VALUE'", arg))
			continue
		}
		err := ret.handleArg(strings.ToLower(strings.TrimSpace(arg[:pos])), arg[pos+1:])
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, startIndex, errors.Join(errs...)
	}
	return ret, startIndex, nil
}

func (o *imageOptions) handleArg(key, value string) error {
	switch key {
	case "image":
		fallthrough
	case "img":
		o.image = value

	case "auth-file":
		o.conn.AuthFilePath = &value

	case "credentials":
		fallthrough
	case "creds":
		o.conn.Creds = &value

	case "username":
		fallthrough
	case "user":
		o.conn.UserName = &value

	case "password":
		fallthrough
	case "pass":
		o.conn.Password = &value

	case "registry-token":
		fallthrough
	case "token":
		o.conn.RegistryToken = &value

	case "certificate-dir":
		fallthrough
	case "cert-path":
		o.conn.DockerCertPath = &value

	case "tls-verify":
		val, err := parseBool(value)
		if err != nil {
			return err
		}
		o.conn.TlsVerify = &val

	case "anonymous":
		fallthrough
	case "no-credentials":
		val, err := parseBool(value)
		if err != nil {
			return err
		}
		o.conn.NoCreds = val

	case "blobs-dir":
		o.conn.SharedBlobsDir = &value

	case "daemon-host":
		o.conn.DockerDaemonHost = &value

	case "include-contents":
		val, err := parseBool(value)
		if err != nil {
			return err
		}
		o.includeContents = val

	case "manifest-size":
		val, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		if val < 0 {
			return fmt.Errorf("bad manifest size (%s); must be non-negative", value)
		}
		o.size = uint64(val)

	case "annotation":
		pos := strings.IndexRune(value, ':')
		if pos < 0 {
			return fmt.Errorf("bad annotation format (%s); expected KEY:VALUE format", value)
		}
		if o.annotations == nil {
			o.annotations = make(map[string]string)
		}
		o.annotations[value[:pos]] = value[pos+1:]

	case "arch":
		fallthrough
	case "architecture":
		o.architecture = value

	case "os":
		o.os = value

	case "os-version":
		o.osVersion = value

	case "os-feature":
		if o.osFeatures == nil {
			o.osFeatures = make([]string, 0)
		}
		o.osFeatures = append(o.osFeatures, value)

	case "os-variant":
		o.variant = value

	case "tag":
		if o.tags == nil {
			o.tags = make([]string, 0)
		}
		o.tags = append(o.tags, value)

	default:
		return fmt.Errorf("invalid image setting (%s)", key)
	}
	return nil
}

func (o *imageOptions) asSource(ctx context.Context) (imgoin.SourceReference, error) {
	if len(o.tags) > 0 {
		return nil, fmt.Errorf("source images cannot take the 'tag' value")
	}

	res := imgoin.AsSourceManifestReference(o.image)
	if res != nil {
		if o.size == 0 {
			return nil, fmt.Errorf("explicitly constructed manifests must include the 'manifest-size' value")
		}

		res.Annotations = o.annotations
		res.Size = o.size
		res.Platform.Architecture = o.architecture
		res.Platform.OS = o.os
		res.Platform.OSVersion = o.osVersion
		res.Platform.OSFeatures = o.osFeatures
		res.Platform.Variant = o.variant
		return res, nil
	}
	img, err := imgoin.AsBearingImage(imgoin.ImageConnection{
		ImageUri:   o.image,
		Connection: &o.conn,
	})
	if err != nil {
		return nil, err
	}
	// TODO this needs to include information on whether the source should also copy over its layers
	return img.AsSourceImage(ctx, o.includeContents)
}

func (o *imageOptions) asTarget(ctx context.Context) (imgoin.TargetImage, error) {
	if o.includeContents {
		return nil, fmt.Errorf("'include-contents' does not belong")
	}
	out, err := imgoin.AsBearingImage(imgoin.ImageConnection{
		ImageUri:   o.image,
		Tags:       o.tags,
		Connection: &o.conn,
	})
	if err != nil {
		return nil, err
	}
	return out.AsTargetImage(ctx)
}
