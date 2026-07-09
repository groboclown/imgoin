#SPDX:Apache-2.0

GO = go
CYCLONEDX = cyclonedx-gomod
GOVULNCHECK = govulncheck
GO_BUILD_FLAGS =
SBOM_FLAGS = -licenses=true -json=true -std=true

SUPPORTED_PLATFORMS := linux-arm64 linux-amd64 darwin-arm64 windows-amd64

OUTDIR := build
BINNAME := imgoin
ifeq ($(findstring Win,$(OS)),Win)
	BINEXT := .exe
else
	BINEXT :=
endif

SOURCE_FILES = $(wildcard *.go pkg/*.go pkg/fixtures/*.go)

## dev                Run the standard development tasks.
.PHONY: dev
dev:

## clean              Remove created files.
.PHONY: clean
clean:

## all                Run the full release tasks.
.PHONY: all
all: clean

dev: $(OUTDIR)/$(BINNAME)$(BINEXT)
$(OUTDIR)/$(BINNAME)$(BINEXT): $(OUTDIR) $(SOURCE_FILES)
	@echo "DEBUG SOURCES: $(SOURCE_FILES)"
	@echo "DEBUG OS: $(OS)"
	$(GO) build -o $@

## test               Run unit tests.
.PHONY: test
dev: test
all: test
test:
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


## go-dependencies    Install required go-based dependencies.
.PHONY: go-dependencies
all: go-dependencies


## format             Reformat the code using Go rules.
.PHONY: format
dev: format
format: $(SOURCE_FILES)
	$(GO) mod fmt ./...


.PHONY: go-dep-vulncheck
go-dependencies: go-dep-vulncheck
go-dep-vulncheck: $(SOURCE_FILES)
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: go-dep-cyclonedx
go-dependencies: go-dep-cyclonedx
go-dep-cyclonedx:
	$(GO) install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest



$(OUTDIR):
	mkdir -p $(OUTDIR)


## Parameterize the per-platform execution.
getOs = $(firstword $(subst -, ,$(1)))
getArch = $(word 2,$(subst -, ,$(1)))
getExt = $(subst windows,.exe,$(findstring windows,$(1)))

define OSBuild =
$(info Supporting $(call getOs,$(1))-$(call getArch,$(1)))

all-binaries: $(OUTDIR)/$(BINNAME)-$(1)$(call getExt,$(1))
$(OUTDIR)/$(BINNAME)-$(1)$(call getExt,$(1)): $(OUTDIR) $(SOURCE_FILES)
	GOOS=$(call getOs,$(1)) GOARCH=$(call getArch,$(1)) $(GO) build -o $$@

all-sboms: $(OUTDIR)/$(BINNAME)-$(1).sbom.json
$(OUTDIR)/$(BINNAME)-$(1).sbom.json: go.mod go.sum
	GOOS=$(call getOs,$(1)) GOARCH=$(call getArch,$(1)) $(CYCLONEDX) app -main . $(SBOM_FLAGS) -output $$@

.PHONY: clean-$(1)
clean: clean-$(1)
clean-$(1):
	-rm -f $(OUTDIR)/$(BINNAME)-$(1)$(call getExt,$(1))
	-rm -f $(OUTDIR)/$(BINNAME)-$(1).sbom.json

endef

$(foreach plat,$(SUPPORTED_PLATFORMS),$(eval $(call OSBuild,$(plat))))
