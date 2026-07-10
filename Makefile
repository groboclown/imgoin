#SPDX:Apache-2.0

GO = go
CYCLONEDX = cyclonedx-gomod
GOVULNCHECK = govulncheck
RM = rm -f
RRM = rm -rf
MKDIR = mkdir -p
CP = cp
ZIPF = zip -9jD

GO_BUILD_FLAGS =
SBOM_FLAGS = -licenses=true -json=true -std=true

SUPPORTED_PLATFORMS := linux-arm64 linux-amd64 darwin-arm64 windows-amd64

OUTDIR := build
DISTDIR := $(OUTDIR)/distribution
BINNAME := imgoin
BINEXT := $(subst Win,.exe,$(findstring Win,$(OS)))

SOURCE_FILES = $(wildcard *.go pkg/*.go) go.mod go.sum
TEST_FILES = $(wildcard pkg/*_test.go pkg/fixtures/*)

## dev                Run the standard development tasks.
.PHONY: dev
dev: format

## clean              Remove created files.
.PHONY: clean
clean:
	-$(RM) $(BINNAME)$(BINEXT)
	-$(RRM) $(DISTDIR)

## all                Run the full release tasks.
.PHONY: all
all: clean

## build              Build the binary for your current platform.
##                    Places the binary at the root of the project directory,
##                    for easy consumption.
.PHONY: build
dev: build
build: $(BINNAME)$(BINEXT)
$(BINNAME)$(BINEXT): $(OUTDIR)/ $(SOURCE_FILES)
	$(GO) build -o $@

## test               Run unit tests.
.PHONY: test
dev: test
all: test
test: $(TEST_FILES) $(SOURCE_FILES)
	$(GO) test ./...


## vulncheck          Check the code for use of vulnerable dependencies.
.PHONY: vulncheck
dev: vulncheck
all: vulncheck
vulncheck: $(SOURCE_FILES)
	$(GOVULNCHECK) ./...


## all-binaries       Generate all platform binaries.
.PHONY: all-binaries
all: all-binaries


## all-sboms          Generate all supported platform SBOMs.
.PHONY: all-sboms
all: all-sboms


## distribution       Create all the files included in the distribution.
.PHONY: distribution
distribution: distribution-bin

.PHONY: distribution-bin
distribution-bin:


## go-dependencies    Install required go-based dependencies.
.PHONY: go-dependencies
all: go-dependencies


## format             Reformat the code using Go rules.
.PHONY: format
format: $(SOURCE_FILES)
	$(GO) fmt ./...


.PHONY: go-dep-vulncheck
go-dependencies: go-dep-vulncheck
go-dep-vulncheck: $(SOURCE_FILES)
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: go-dep-cyclonedx
go-dependencies: go-dep-cyclonedx
go-dep-cyclonedx:
	$(GO) install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest



$(OUTDIR)/:
	-$(MKDIR) $@

$(DISTDIR)/: $(OUTDIR)/
	-$(MKDIR) $@

$(OUTDIR)/LICENSE: LICENSE $(OUTDIR)/
	$(CP) LICENSE $@

# Parameterize the per-platform execution.
#
# This uses a macro to construct the targets for the 'all-binaries' and
# 'all-sboms' and 'clean' targets, one for each supported platform.
# Without this, the build would include many cut-and-paste targets.
getOs = $(firstword $(subst -, ,$(1)))
getArch = $(word 2,$(subst -, ,$(1)))
getExt = $(subst windows,.exe,$(findstring windows,$(1)))

define OSBuild =
$(info Supporting $(call getOs,$(1))-$(call getArch,$(1)))


all-binaries: $(OUTDIR)/$(BINNAME)-$(1)$(call getExt,$(1))
$(OUTDIR)/$(BINNAME)-$(1)$(call getExt,$(1)): $(OUTDIR)/ $(SOURCE_FILES)
	GOOS=$(call getOs,$(1)) GOARCH=$(call getArch,$(1)) $(GO) build -o $$@

all-sboms: $(OUTDIR)/$(BINNAME)-$(1).sbom.json
$(OUTDIR)/$(BINNAME)-$(1).sbom.json: go.mod go.sum
	GOOS=$(call getOs,$(1)) GOARCH=$(call getArch,$(1)) $(CYCLONEDX) app -main . $(SBOM_FLAGS) -output $$@

.PHONY: clean-$(1)
clean: clean-$(1)
clean-$(1):
	-$(RM) $(OUTDIR)/$(BINNAME)-$(1)$(call getExt,$(1))
	-$(RM) $(OUTDIR)/$(BINNAME)-$(1).sbom.json

distribution-bin: $(DISTDIR)/$(BINNAME)-$(1)$(call getExt,$(1))
$(DISTDIR)/$(BINNAME)-$(1)$(call getExt,$(1)): $(OUTDIR)/$(BINNAME)-$(1)$(call getExt,$(1)) $(DISTDIR)/
	$(CP) $$< $$@

distribution-bin: $(DISTDIR)/$(BINNAME)-$(1).zip
$(DISTDIR)/$(BINNAME)-$(1).zip: $(OUTDIR)/$(BINNAME)-$(1)$(call getExt,$(1)) $(OUTDIR)/$(BINNAME)-$(1).sbom.json $(OUTDIR)/LICENSE $(DISTDIR)/
	$(ZIPF) $$@ $$^


endef

$(foreach plat,$(SUPPORTED_PLATFORMS),$(eval $(call OSBuild,$(plat))))
