// SPDX:Apache-2.0
package cmd

import (
	"fmt"

	"go.podman.io/image/v5/transports"
)

func printHelp(name string) {
	fmt.Printf(`Usage: %s [-h] --source SETTINGS [--source SETTINGS ...] --target SETTINGS
Where:
   -h, --help       This screen.
   --source         Settings for the source image.  You may specify
                    multiple source images.
   --target			Settings for the target image.

Each '--source' and '--target' argument tells the command to start reading
information about that corresponding source or target.

The SETTINGS takes the form of 'KEY=VALUE'; see the below table for the list
of supported keys and their recognized values.  Some keys allow for setting
multiple values.

Alternatively, you can pass '@FILENAME' to have the command read the settings
from the file named FILENAME.  This file contains 'KEY=VALUE' items, separated
by spaces or newlines.  You can put the value in quotes if it contains a
space, or use \\ to escape a character (like a quote).

General image settings:
  image=IMAGE-LOCATION
      Location of the image, in the format 'transport:name' (see the list of
	  supported transports below).
	  Required for all images.
      Aliases: 'img'.

  auth-file=FILENAME
      Path of the registry credentials file.
      Default is ${XDG_RUNTIME_DIR}/containers/auth.json

  credentials=USERNAME[:PASSWORD]
      Credentials for accessing the registry.
      Cannot be used with the username setting.
      Aliases: 'creds'.

  username=USERNAME
      Username for accessing the registry.
      Cannot be used with the credentials setting.
      If given, you must also provide the password setting, even if empty.
      Aliases: 'user'.

  password=PASSWORD
      Password for accessing the registry.
      Must be used with the 'username' setting.
      Aliases: 'pass'.

  anonymous=yes|no
      Set to 'yes' to force anonymous registry access.
      This overrides username or credentials settings.
      Aliases: 'no-credentials'.

  registry-token=TOKEN
      Provides a bearer token for accessing the registry.
      Aliases: 'token'.

  certificate-dir=DIRNAME
      Directory containing *.{crt,cert,key} files for contacting the registry.
      Aliases: 'cert-path'.

  skip-tls-verify=yes|no
      Set to 'yes' to force ignoring HTTPS + certificate verification.

  tls-details=FILENAME
      Path to a containers-tls-details.yaml(5) file.

  user-agent-prefix=PREFIX
      Prefix to add to the user agent string.

  registries-conf=FILENAME
      Path to the registries.conf file.

  registries.d=DIR
      Use registry configuration files in 'DIR' (e.g. for container signature storage).

  blobs-dir=DIRNAME
      DIR to use to share blobs across OCI repositories.

  daemon-host=HOSTNAME[:PORT]
      Use docker daemon host at HOSTNAME (docker-daemon: transports only).

  tmp-dir=DIR
      Path to use for big temporary files.

Source image settings:

  include-contents=yes|no
      Set to 'yes' to have the target image save all the contents of this
	  source image.  Without it, the target image will store just a reference.
	  Most multi-architecture images store just a reference.

  policy=FILE
      Use a signature policy file at the given path.  Used for checking the
      source images.

  insecure-policy=yes|no
      Set to 'yes' to force the program to ignore signatures on source images.

  require-signed=yes|no
      Set to 'yes' for require all source images to have a valid signature.
      Source images without a signature will cause a failure.

Explicit source image settings:

  You can pass the image setting in the form 'image=sha256:ABC...', which
  allows you to explicitly declare the image reference information.

  manifest-size=SIZE_IN_BYTES
      Number of bytes of the referenced image's manifest.
	
  architecture=ARCH
      Name of the image's target architecture.
      Aliases: 'arch'.
  
  os=OS
      Name of the image's target operating system.

  os-version=VERSION
      Version of the image's target operating system.

  os-feature=FEATURE
      A feature required for the image's target operating system.
      You may specify this more than once.

  os-variant=VARIANT
      The image's target operating system variant.

  annotation=KEY:VALUE
      Add a annotation to the image reference.
	  You may provide multiple of these.

Target image settings:

  tag=TAG
      Give the constructed image a tag.  Only some output transports support this.
      You may provide multiple of these.

Supported image transports:
`, name)
	for _, name := range transports.ListNames() {
		fmt.Printf("     %s\n", name)
	}
}

func printVersion(version string) {
	fmt.Printf("imgoin v%s\n", version)
}
