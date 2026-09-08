#SPDX:Apache-2.0

GO = go
CYCLONEDX = cyclonedx-gomod
GOVULNCHECK = govulncheck
MKDIR = mkdir -p
RMDIR = rmdir
CP = cp
ZIPF = zip -9jD

BTRFS_TAG := $(shell CC=$(CC) CFLAGS=$(CFLAGS) ./make_scripts/detect-btrfs.sh )
ENCRYPT_TAG = containers_image_openpgp
GO_TAGS = $(BTRFS_TAG) $(ENCRYPT_TAG)

GO_BUILD_FLAGS =
GO_TAG_FLAGS = -tags '$(GO_TAGS)'
SBOM_FLAGS = -licenses=true -json=true -std=true

ALL_GO_PLATFORMS := $(subst /,-,$(shell $(GO) tool dist list))
LINUX_PLATFORMS := $(filter-out %.loong64,$(filter linux-%,$(ALL_GO_PLATFORMS)))
DARWIN_PLATFORMS := $(filter darwin-%,$(ALL_GO_PLATFORMS))
WINDOWS_PLATFORMS := $(filter windows-%,$(ALL_GO_PLATFORMS))
FREEBSD_PLATFORMS := $(filter freebsd-%,$(ALL_GO_PLATFORMS))
SUPPORTED_PLATFORMS = $(LINUX_PLATFORMS) $(DARWIN_PLATFORMS) $(WINDOWS_PLATFORMS) $(FREEBSD_PLATFORMS)

OUTDIR := build
DISTDIR := $(OUTDIR)/distribution
CROSS_BINDIR := $(OUTDIR)/cross-bin
CROSS_SBOMDIR := $(OUTDIR)/cross-sbom
BINNAME := imgoin
BINEXT := $(subst Win,.exe,$(findstring Win,$(OS)))

SOURCE_FILES = $(wildcard *.go pkg/*.go) go.mod go.sum
TEST_FILES = $(wildcard pkg/*_test.go pkg/fixtures/*)


## dev                Run the standard development tasks.
##
.PHONY: dev
dev: format

## clean              Remove created files.
##
.PHONY: clean

## all                Run the full release tasks.  Does not clean.
##
.PHONY: all
all:

## build              Build the binary for your current platform.
##                    Places the binary at the root of the project directory,
##                    for easy consumption.
##
.PHONY: build
dev: build
build: $(BINNAME)$(BINEXT)
$(BINNAME)$(BINEXT): $(OUTDIR)/ $(SOURCE_FILES)
	$(GO) build $(GO_TAG_FLAGS) $(GO_BUILD_FLAGS) -o $@

clean: clean-build
clean-build:
	-$(RM) $(BINNAME)$(BINEXT)


## test               Run unit tests.
##
.PHONY: test
dev: test
all: test
test: $(TEST_FILES) $(SOURCE_FILES)
	 $(GO) test $(GO_BUILD_FLAGS) ./...


## sbom               Generate the SBOM for the current platform.
##
.PHONY: sbom
sbom: $(OUTDIR)/$(BINNAME).sbom.json
$(OUTDIR)/$(BINNAME).sbom.json: $(OUTDIR)/ go.mod go.sum
	$(CYCLONEDX) app -main . $(SBOM_FLAGS) -output $@

clean: clean-sbom
clean-sbom:
	-$(RM) $(OUTDIR)/$(BINNAME).sbom.json


## vulncheck          Check the code for use of vulnerable dependencies.
##                    Stops the build on a discovered vulnerability.
##
.PHONY: vulncheck
dev: vulncheck
all: vulncheck
# This will fail due to issues with gpgme on Linux if CGO_ENABLED=0
vulncheck: $(SOURCE_FILES) $(TEST_FILES)
	$(GOVULNCHECK) $(GO_TAG_FLAGS) ./...


## vulncheck-verbose  Check the code for use of vulnerable dependencies.
##                    It stops the build on a discovered vulnerability.
##
.PHONY: vulncheck-verbose
vulncheck-verbose: $(SOURCE_FILES) $(TEST_FILES)
	$(GOVULNCHECK) -show verbose $(GO_TAG_FLAGS) ./...


## all-binaries       Generate all platform binaries.
##
.PHONY: all-binaries
all: all-binaries
WINDOWS_BIN_TARGETS=$(addsuffix .exe,$(filter windows-%,$(SUPPORTED_PLATFORMS)))
OTHER_BIN_TARGETS=$(filter-out windows-%,$(SUPPORTED_PLATFORMS))
ALL_CROSS_TARGETS=$(addprefix $(CROSS_BINDIR)/$(BINNAME).,$(subst -,.,$(WINDOWS_BIN_TARGETS) $(OTHER_BIN_TARGETS)))

all-binaries: $(ALL_CROSS_TARGETS)
$(CROSS_BINDIR)/$(BINNAME).%: $(CROSS_BINDIR)/ $(SOURCE_FILES)
	GOOS=$(word 2,$(subst ., ,$@)) GOARCH=$(word 3,$(subst ., ,$@)) CGO_ENABLED=0 $(GO) build -tags '$(ENCRYPT_TAG)' -o $@

clean: clean-cross-bin
clean-cross-bin:
	-$(RM) $(ALL_CROSS_TARGETS)


## all-sboms          Generate all supported platform SBOMs.
##
.PHONY: all-sboms
all: all-sboms
ALL_SBOM_TARGETS=$(addprefix $(CROSS_SBOMDIR)/$(BINNAME).,$(addsuffix .sbom.json,$(subst -,.,$(SUPPORTED_PLATFORMS))))

all-sboms: $(ALL_SBOM_TARGETS)
$(CROSS_SBOMDIR)/$(BINNAME).%: $(CROSS_SBOMDIR)/ go.mod go.sum
	GOOS=$(word 2,$(subst ., ,$@)) GOARCH=$(word 3,$(subst ., ,$@)) CGO_ENABLED=0 $(CYCLONEDX) app -main . $(SBOM_FLAGS) -output $@

clean: clean-cross-sbom
clean-cross-sbom:
	-$(RM) $(ALL_SBOM_TARGETS)


## distribution       Create all the files included in the distribution.
##
.PHONY: distribution
all: distribution
ALL_DIST_TARGETS=$(addprefix $(DISTDIR)/$(BINNAME).,$(addsuffix .zip,$(subst -,.,$(SUPPORTED_PLATFORMS))))

distribution: $(ALL_DIST_TARGETS)
$(DISTDIR)/$(BINNAME).%: $(ALL_CROSS_TARGETS) $(ALL_SBOM_TARGETS) LICENSE $(DISTDIR)/ $(OUTDIR)
	$(MKDIR) $@.d
	$(CP) \
		$(CROSS_BINDIR)/$(BINNAME).$(word 2,$(subst ., ,$@)).$(word 3,$(subst ., ,$@))$(subst $(word 2,$(subst ., ,$@)),,$(subst windows,.exe,$(word 2,$(subst ., ,$@)))) \
		$@.d/$(BINNAME)$(subst $(word 2,$(subst ., ,$@)),,$(subst windows,.exe,$(word 2,$(subst ., ,$@))))
	$(ZIPF) $@ LICENSE \
		$(CROSS_SBOMDIR)/$(BINNAME).$(word 2,$(subst ., ,$@)).$(word 3,$(subst ., ,$@)).sbom.json \
		$@.d/$(BINNAME)$(subst $(word 2,$(subst ., ,$@)),,$(subst windows,.exe,$(word 2,$(subst ., ,$@))))
	$(RM) $@.d/$(BINNAME)$(subst $(word 2,$(subst ., ,$@)),,$(subst windows,.exe,$(word 2,$(subst ., ,$@))))
	$(RMDIR) $@.d

clean: clean-dist
clean-dist:
	-$(RM) $(ALL_DIST_TARGETS) $(OUTDIR)/$(BINNAME) $(OUTDIR)/$(BINNAME).exe


## format             Reformat the code using Go rules.
##
.PHONY: format
format: $(SOURCE_FILES)
	$(GO) fmt ./...


## go-dependencies    Install required go-based dependencies.
##
.PHONY: go-dependencies
go-dependencies:

.PHONY: go-dep-vulncheck
go-dependencies: go-dep-vulncheck
go-dep-vulncheck: $(SOURCE_FILES)
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: go-dep-cyclonedx
go-dependencies: go-dep-cyclonedx
go-dep-cyclonedx:
	$(GO) install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest


## help               This screen.
##
.PHONY: help
help:
	@grep -E '^##' $(lastword $(MAKEFILE_LIST)) | cut -c 4-


$(DISTDIR)/: $(OUTDIR)/
	-$(MKDIR) $@

$(CROSS_BINDIR)/: $(OUTDIR)/
	-$(MKDIR) $@

$(CROSS_SBOMDIR)/: $(OUTDIR)/
	-$(MKDIR) $@

$(OUTDIR)/:
	-$(MKDIR) $@
