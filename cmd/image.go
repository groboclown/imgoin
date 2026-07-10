// SPDX:Apache-2.0
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	imgoin "github.com/groboclown/imgoin/pkg"
	"go.podman.io/image/v5/pkg/cli/basetls/tlsdetails"
	"go.podman.io/image/v5/types"
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

	// Collects the connection information.
	sys *types.SystemContext

	// Additional information for joined-together understanding of the system context.
	creds          *string // 'username[:password]' for accessing the registry
	userName       *string // username for accessing the registry
	password       *string // password for accessing the registry
	noCreds        bool
	skipTlsVerify  *bool
	tlsDetailsPath string

	// If true, all the contents of this source image are copied into the target.
	includeContents bool

	// Targets can also include different tags.
	tags []string

	baseUserAgent string
}

// parseImageArgs reads in the arguments for an image, starting with the given index.
// It reads in the 'KEY=VALUE' argument formats.  It stops at the end of the argument list
// or at the first argument starting with '-'.  It returns the index it stopped reading at.
func parseImageArgs(version string, args []string, startIndex int) (*imageOptions, int, error) {
	ret := defaultImageOptions(version)
	idx, err := ret.handleArgs(args, startIndex)
	return ret, idx, err
}

// Create the default setup for an image.
func defaultImageOptions(version string) *imageOptions {
	userAgent := "imgion/" + version
	return &imageOptions{
		baseUserAgent: userAgent,
		sys: &types.SystemContext{
			DockerRegistryUserAgent: userAgent,
			BigFilesTemporaryDir:    os.Getenv("TMPDIR"),
			AuthFilePath:            os.Getenv("REGISTRY_AUTH_FILE"),
		},
		tags: make([]string, 0),
	}
}

// parseImageOptions reads in the options for the image, starting with the given index.
// It reads in the 'KEY=VALUE' argument formats.  It stops at the end of the argument list
// or at the first argument starting with '-'.  It returns the index it stopped reading at.
func (o *imageOptions) handleArgs(args []string, startIndex int) (int, error) {
	if o == nil {
		panic("BUG bad usage; self must not be nil")
	}

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
		if arg[0] == '@' {
			// Read the options from the file instead.
			fileArgs, err := SplitFileArgs(arg[1:])
			if err != nil {
				errs = append(errs, err)
			} else {
				for _, a := range fileArgs {
					errs = append(errs, o.handleArg(a))
				}
			}
			continue
		}

		// Read as key=value format.
		errs = append(errs, o.handleArg(arg))
	}
	return startIndex, errors.Join(errs...)
}

func (o *imageOptions) handleArg(arg string) error {
	pos := strings.IndexRune(arg, '=')
	if pos <= 0 {
		return fmt.Errorf("invalid image argument (%s); must be in the format 'KEY=VALUE'", arg)
	}
	return o.handleSetting(strings.ToLower(strings.TrimSpace(arg[:pos])), arg[pos+1:])
}

func (o *imageOptions) handleSetting(key, value string) error {
	switch key {

	// transport + image name.
	// Required.
	case "image":
		fallthrough
	case "img":
		o.image = value

	// ----------------------------------
	// Authentication stuff.

	// Path of the registry credentials file. Default is ${XDG_RUNTIME_DIR}/containers/auth.json
	case "auth-file":
		o.sys.AuthFilePath = value

	// Pass a username[:password] to the registry.
	// Cannot be used with username, password, or anonymous.
	case "credentials":
		fallthrough
	case "creds":
		o.creds = &value

	// Pass a username to the registry.
	// Cannot be used with credentials or anonymous.
	case "username":
		fallthrough
	case "user":
		o.userName = &value

	// Pass a password to the registry.
	// Cannot be used with credentials or anonymous.
	case "password":
		fallthrough
	case "pass":
		o.password = &value

	// Explicitly do not pass login information to the registry.
	// Cannot be used with credentials, username, or password.
	case "anonymous":
		fallthrough
	case "no-credentials":
		val, err := parseBool(value)
		if err != nil {
			return err
		}
		o.noCreds = val

	// Provide a Bearer token for accessing the registry.
	case "registry-token":
		fallthrough
	case "token":
		o.sys.DockerBearerRegistryToken = value

	// ----------------------------------
	// Note: currently not needing a signature.PolicyContext
	// because this doesn't yet support signatures.

	// Path to a trust policy file
	case "policy":
		return fmt.Errorf("signature policies not supported")

	// Run the tool without any policy check
	case "insecure-policy":
		// Does nothing.

	// Require any pulled image to be signed
	case "require-signed":
		return fmt.Errorf("signatures not supported")

	// ----------------------------------
	// Connection information

	// Use certificates at `PATH` (*.crt, *.cert, *.key) to connect to the registry or daemon
	case "certificate-dir":
		fallthrough
	case "cert-path":
		o.sys.DockerCertPath = value
		o.sys.DockerDaemonCertPath = value

	// Don't require HTTPS and verify certificates (for docker: and docker-daemon:)
	case "skip-tls-verify":
		val, err := parseBool(value)
		if err != nil {
			return err
		}
		o.skipTlsVerify = &val
		o.sys.DockerDaemonInsecureSkipTLSVerify = val
		o.sys.DockerInsecureSkipTLSVerify = types.NewOptionalBool(val)

	// Path to a containers-tls-details.yaml(5) file
	case "tls-details":
		o.tlsDetailsPath = value

	// Prefix to add to the user agent string
	case "user-agent-prefix":
		value = strings.TrimSpace(value)
		if value != "" {
			o.sys.DockerRegistryUserAgent = value + " " + o.baseUserAgent
		}

	// Path to the registries.conf file
	case "registries-conf":
		o.sys.SystemRegistriesConfPath = value

	// Use registry configuration files in `DIR` (e.g. for container signature storage)
	case "registries.d":
		o.sys.RegistriesDirPath = value

	// `DIR` to use to share blobs across OCI repositories
	case "blobs-dir":
		o.sys.OCISharedBlobDirPath = value

	// Use docker daemon host at `HOST` (docker-daemon: only)
	case "daemon-host":
		o.sys.DockerDaemonHost = value

	// ----------------------------------
	// Explicit image generation.

	// Set the source image's referenced manifest size.
	case "manifest-size":
		val, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		if val < 0 {
			return fmt.Errorf("bad manifest size (%s); must be non-negative", value)
		}
		o.size = uint64(val)

	// Add an annotation to the source image.
	case "annotation":
		pos := strings.IndexRune(value, ':')
		if pos < 0 {
			return fmt.Errorf("bad annotation format (%s); expected KEY:VALUE format", value)
		}
		if o.annotations == nil {
			o.annotations = make(map[string]string)
		}
		o.annotations[value[:pos]] = value[pos+1:]

	// Set the source image's architecture.
	case "arch":
		fallthrough
	case "architecture":
		o.architecture = value

	// Set the source image's operating system.
	case "os":
		o.os = value

	// Set the source image's operating system version.
	case "os-version":
		o.osVersion = value

	// Add an operating system feature to the source image.  May have multiple of these.
	case "os-feature":
		if o.osFeatures == nil {
			o.osFeatures = make([]string, 0)
		}
		o.osFeatures = append(o.osFeatures, value)

	// Set the source image's operating system variant name.
	case "os-variant":
		o.variant = value

	// ----------------------------------
	// Other stuff

	// Path to use for big temporary files
	case "tmp-dir":
		o.sys.BigFilesTemporaryDir = value

	case "include-contents":
		val, err := parseBool(value)
		if err != nil {
			return err
		}
		o.includeContents = val

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
	sys, err := o.finalizeSys()
	if err != nil {
		return nil, err
	}
	img, err := imgoin.AsBearingImage(imgoin.ImageConnection{
		ImageUri: o.image,
		System:   sys,
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
	sys, err := o.finalizeSys()
	if err != nil {
		return nil, err
	}
	out, err := imgoin.AsBearingImage(imgoin.ImageConnection{
		ImageUri: o.image,
		Tags:     o.tags,
		System:   sys,
	})
	if err != nil {
		return nil, err
	}
	return out.AsTargetImage(ctx)
}

func (o *imageOptions) finalizeSys() (*types.SystemContext, error) {
	baseTLSConfig, err := tlsdetails.BaseTLSFromOptionalFile(o.tlsDetailsPath)
	if err != nil {
		return nil, err
	}
	o.sys.BaseTLSConfig = baseTLSConfig.TLSConfig()
	if o.skipTlsVerify != nil {
		o.sys.DockerInsecureSkipTLSVerify = types.NewOptionalBool(*o.skipTlsVerify)
	}

	if o.creds != nil {
		if o.userName != nil || o.password != nil {
			return nil, fmt.Errorf("Cannot specify both 'credentials' and 'username' settings")
		}
		if o.noCreds {
			return nil, fmt.Errorf("Cannot specify both 'credentials' and 'anonymous' settings")
		}
		user, pass, err := parseCreds(*o.creds)
		if err != nil {
			return nil, err
		}
		o.sys.DockerAuthConfig = getDockerAuth(user, pass)
	} else if o.userName != nil {
		if o.noCreds {
			return nil, fmt.Errorf("Cannot specify both 'username' and 'anonymous' settings")
		}
		if o.password == nil {
			return nil, fmt.Errorf("'password' must be set because 'username' is set")
		}
		o.sys.DockerAuthConfig = getDockerAuth(*o.userName, *o.password)
	} else if o.noCreds {
		o.sys.DockerAuthConfig = &types.DockerAuthConfig{}
	}

	return o.sys, nil
}

func parseCreds(creds string) (string, string, error) {
	if creds == "" {
		return "", "", errors.New("credentials can't be empty")
	}
	username, password, _ := strings.Cut(creds, ":") // Sets password to "" if there is no ":"
	if username == "" {
		return "", "", errors.New("username can't be empty")
	}
	return username, password, nil
}

func getDockerAuth(username, password string) *types.DockerAuthConfig {
	return &types.DockerAuthConfig{
		Username: username,
		Password: password,
	}
}
