// SPDX:Apache-2.0
package imgoin

import (
	"context"
	"fmt"
	"io"
	"os"

	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/transports/alltransports"
	"go.podman.io/image/v5/types"
)

// ImageConnection points to a defined image somewhere, using a supported podman image transport.
type ImageConnection struct {
	ImageUri   string
	Connection *RepositoryConnection
}

type SourceReference interface {
	GetManifests(ctx context.Context) ([]manifest.ListUpdate, error)
	GetBlobsToCopy(ctx context.Context) ([]types.BlobInfo, error)
	GetBlob(ctx context.Context, digest types.BlobInfo) (io.ReadCloser, error)
	Close() error
}

type TargetImage interface {
	AddManifest(manifest.ListUpdate) error
	CopyBlob(types.BlobInfo, io.ReadCloser) error
	Close() error
}

// SourceManifestReference allows for specifying the meta-data information explicitly.
// This points to the *manifest* information, not that manifest's blob.
// The alternative references the image using the podman images API.
// This can only reference a source manifest, as the target must generate a real thing.
type SourceManifestReference struct {
	Digest      digest.Digest
	Size        uint64
	Annotations map[string]string
	Platform    *imgspecv1.Platform
}

// AsSourceManifestReference constructs a SourceManifestReference based on the image name.
// It does not populate the size, annotations, or platform.
// If the `imageName` does not use a digest prefix, then this returns nil.
func AsSourceManifestReference(imageName string) *SourceManifestReference {
	digest, err := digest.Parse(imageName)
	if err != nil {
		return nil
	}
	return &SourceManifestReference{
		Digest: digest,
		Platform: &imgspecv1.Platform{
			OSFeatures: make([]string, 0),
		},
	}
}

// BearingImage references an image that contains either the manifest index or the data blob.
type BearingImage struct {
	sys *types.SystemContext
	ref types.ImageReference
}

// RepositoryConnection contains information on how to contact the registry.
type RepositoryConnection struct {
	AuthFilePath     *string // Path to a 'containers/auth.json' file.
	Creds            *string // 'username[:password]' for accessing the registry
	UserName         *string // username for accessing the registry
	Password         *string // password for accessing the registry
	RegistryToken    *string // bearer token-like access to the registry
	DockerCertPath   *string // Directory containing *.{crt,cert,key} files for contacting the registry.
	TlsVerify        *bool   // Require, don't require, or default for HTTPS + certificate verification.
	NoCreds          bool    // Force anonymous registry access.
	SharedBlobsDir   *string // OCI shared blobs directory.
	DockerDaemonHost *string // docker-daemon host for the connection.
}

// AsBearingImage constructs a BearingImage from an image connection.
func AsBearingImage(conn ImageConnection) (*BearingImage, error) {
	if conn.Connection == nil {
		return nil, fmt.Errorf("BUG nil connection information")
	}
	ref, err := alltransports.ParseImageName(conn.ImageUri)
	if err != nil {
		return nil, err
	}

	// Special authentication setup.
	token := ""
	var auth *types.DockerAuthConfig = nil
	if conn.Connection.RegistryToken != nil {
		token = *conn.Connection.RegistryToken
	} else if conn.Connection.UserName != nil {
		auth = &types.DockerAuthConfig{
			Username: *conn.Connection.UserName,
			Password: ptrAsStr(conn.Connection.Password),
			// Not here: IdentityToken
		}
	}

	return &BearingImage{
		sys: &types.SystemContext{
			// If not "", prefixed to any absolute paths used by default by the library (e.g. in /etc/).
			// Not used for any of the more specific path overrides available in this struct.
			// Not used for any paths specified by users in config files (even if the location of the config file _was_ affected by it).
			// NOTE: If this is set, environment-variable overrides of paths are ignored (to keep the semantics simple: to create an /etc replacement, just set RootForImplicitAbsolutePaths .
			// and there is no need to worry about the environment.)
			// NOTE: This does NOT affect paths starting by $HOME.
			RootForImplicitAbsolutePaths: "",

			// === Global configuration overrides ===
			// If not "", overrides the system's default path for signature.Policy configuration.
			// ... This currently doesn't sign
			SignaturePolicyPath: "",
			// If not "", overrides the system's default path for registries.d (Docker signature storage configuration)
			RegistriesDirPath: "",
			// Path to the system-wide registries configuration file
			SystemRegistriesConfPath: "",
			// Path to the system-wide registries configuration directory
			SystemRegistriesConfDirPath: "",
			// Path to the user-specific short-names configuration file
			UserShortNameAliasConfPath: "",
			// If set, short-name resolution in pkg/shortnames must follow the specified mode
			ShortNameMode: nil,
			// If set, short names will resolve in pkg/shortnames to docker.io only, and unqualified-search registries and
			// short-name aliases in registries.conf are ignored.  Note that this field is only intended to help enforce
			// resolving to Docker Hub in the Docker-compatible REST API of Podman; it should never be used outside this
			// specific context.
			PodmanOnlyShortNamesIgnoreRegistriesConfAndForceDockerHub: false,
			// If not "", overrides the default path for the registry authentication file, but only new format files
			AuthFilePath: ptrAsStr(conn.Connection.AuthFilePath),
			// if not "", overrides the default path for the registry authentication file, but with the legacy format;
			// the code currently will by default look for legacy format files like .dockercfg in the $HOME dir;
			// but in addition to the home dir, openshift may mount .dockercfg files (via secret mount)
			// in locations other than the home dir; openshift components should then set this field in those cases;
			// this field is ignored if `AuthFilePath` is set (we favor the newer format);
			// only reading of this data is supported;
			LegacyFormatAuthFilePath: "",
			// If set, a path to a Docker-compatible "config.json" file containing credentials; and no other files are processed.
			// This must not be set if AuthFilePath is set.
			// Only credentials and credential helpers in this file apre processed, not any other configuration in this file.
			DockerCompatAuthFilePath: "",
			// If not "", overrides the use of platform.GOARCH when choosing an image or verifying architecture match.
			ArchitectureChoice: "",
			// If not "", overrides the use of platform.GOOS when choosing an image or verifying OS match.
			OSChoice: "",
			// If not "", overrides the use of detected ARM platform variant when choosing an image or verifying variant match.
			VariantChoice: "",
			// If not "", overrides the system's default directory containing a blob info cache.
			BlobInfoCacheDir: "",
			// Additional tags when creating or copying a docker-archive.
			DockerArchiveAdditionalTags: nil,
			// If not "", overrides the temporary directory to use for storing big files
			BigFilesTemporaryDir: os.Getenv("TMPDIR"),
			// If not nil, may contain TLS _algorithm_ options (e.g. TLS version, cipher suites, “curves”, etc.)
			// The effect of setting any other options (cryptographic keys, InsecureSkipTLSVerify, callbacks, etc.) is UNDEFINED,
			// may be inconsistent in various use cases, and may change over time.
			// Consumers of this value are expected to .Clone() the config and then apply other options.
			BaseTLSConfig: nil,

			// === OCI.Transport overrides ===
			// If not "", a directory containing a CA certificate (ending with ".crt"),
			// a client certificate (ending with ".cert") and a client certificate key
			// (ending with ".key") used when downloading OCI image layers.
			OCICertPath: ptrAsStr(conn.Connection.DockerCertPath),
			// Allow downloading OCI image layers over HTTP, or HTTPS with failed TLS verification. Note that this does not affect other TLS connections.
			OCIInsecureSkipTLSVerify: ptrAsBool(conn.Connection.TlsVerify, false),
			// If not "", use a shared directory for storing blobs rather than within OCI layouts
			OCISharedBlobDirPath: ptrAsStr(conn.Connection.SharedBlobsDir),
			// Allow UnCompress image layer for OCI image layer
			OCIAcceptUncompressedLayers: true,

			// === docker.Transport overrides ===
			// If not "", a directory containing a CA certificate (ending with ".crt"),
			// a client certificate (ending with ".cert") and a client certificate key
			// (ending with ".key") used when talking to a container registry.
			DockerCertPath: ptrAsStr(conn.Connection.DockerCertPath),
			// If not "", overrides the system’s default path for a directory containing host[:port] subdirectories with the same structure as DockerCertPath above.
			// Ignored if DockerCertPath is non-empty.
			DockerPerHostCertDirPath: "",
			// Allow contacting container registries over HTTP, or HTTPS with failed TLS verification. Note that this does not affect other TLS connections.
			DockerInsecureSkipTLSVerify: ptrAsOptionalBool(conn.Connection.TlsVerify),
			// if nil, the library tries to parse ~/.docker/config.json to retrieve credentials
			// Ignored if DockerBearerRegistryToken is non-empty.
			DockerAuthConfig: auth,
			// if not "", the library uses this registry token to authenticate to the registry
			DockerBearerRegistryToken: token,
			// if not "", an User-Agent header is added to each request when contacting a registry.
			DockerRegistryUserAgent: "",
			// If true, dockerImageDestination.SupportedManifestMIMETypes will omit the Schema1 media types from the supported list
			DockerDisableDestSchema1MIMETypes: true,
			// If true, the physical pull source of docker transport images logged as info level
			DockerLogMirrorChoice: true,
			// If true, all blobs will have precomputed digests to ensure layers are not uploaded that already exist on the registry.
			// Note that this requires writing blobs to temporary files, and takes more time than the default behavior,
			// when the digest for a blob is unknown.
			DockerRegistryPushPrecomputeDigests: true,
			// DockerProxyURL specifies proxy configuration schema (like socks5://username:password@ip:port)
			DockerProxyURL: nil,
			// DockerProxy is a function that determines the proxy URL for a given request URL.
			// If set, this takes precedence over DockerProxyURL. The function should return the proxy URL to use,
			// or nil if no proxy should be used for the given request.
			DockerProxy: nil,

			// === docker/daemon.Transport overrides ===
			// A directory containing a CA certificate (ending with ".crt"),
			// a client certificate (ending with ".cert") and a client certificate key
			// (ending with ".key") used when talking to a Docker daemon.
			DockerDaemonCertPath: "",
			// The hostname or IP to the Docker daemon. If not set (aka ""), client.DefaultDockerHost is assumed.
			DockerDaemonHost: ptrAsStr(conn.Connection.DockerDaemonHost),
			// Used to skip TLS verification, off by default. To take effect DockerDaemonCertPath needs to be specified as well.
			DockerDaemonInsecureSkipTLSVerify: false,

			// === dir.Transport overrides ===
			// DirForceCompress compresses the image layers if set to true
			DirForceCompress: false,
			// DirForceDecompress decompresses the image layers if set to true
			DirForceDecompress: false,

			// CompressionFormat is the format to use for the compression of the blobs
			CompressionFormat: nil,
			// CompressionLevel specifies what compression level is used
			CompressionLevel: nil,
		},
		ref: ref,
	}, nil
}

func ptrAsStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrAsBool(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

func ptrAsOptionalBool(b *bool) types.OptionalBool {
	if b == nil {
		return types.OptionalBoolUndefined
	}
	if *b {
		return types.OptionalBoolTrue
	}
	return types.OptionalBoolFalse
}
