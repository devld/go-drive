
work_dir = build
target_name ?= go-drive_build
build_dir = $(work_dir)/$(target_name)
BUILD_REV := $(or $(BUILD_REV),$(shell git rev-parse HEAD 2>/dev/null),unknown)
BUILD_AT := $(or $(BUILD_AT),$(shell date -R),unknown)

all: $(build_dir)/$(target_name).tar.gz
zip: $(build_dir)/$(target_name).zip

# tar.gz
$(build_dir)/$(target_name).tar.gz: $(build_dir)/$(target_name)
	cd $(work_dir); tar acf $(target_name).tar.gz --owner=0 --group=0 $(target_name)

# zip for windows
$(build_dir)/$(target_name).zip: $(build_dir)/$(target_name)
	cd $(work_dir); zip -q -r $(target_name).zip $(target_name)

$(build_dir)/$(target_name): $(build_dir)/go-drive $(build_dir)/config.yml

# The web UI (web/dist) and i18n files (docs/lang) are embedded into the binary,
# so the frontend must be built before linking. The web UI embed only happens
# under the "release" build tag; without it (e.g. `go test`/`go build`) web/dist
# is not required.
$(build_dir)/go-drive: $(build_dir) web/dist
	CGO_CFLAGS="-Wno-return-local-addr" \
	go build -tags release -o $(build_dir) -ldflags \
		"-w -s \
		-X 'go-drive/common.Version=${BUILD_VERSION}' \
		-X 'go-drive/common.RevHash=$(BUILD_REV)' \
		-X 'go-drive/common.BuildAt=$(BUILD_AT)'"

$(build_dir)/config.yml: $(build_dir)
	cp docs/config.yml $(build_dir)/

web/dist:
	cd web; npm install && npm run build

$(build_dir):
	mkdir -p $(build_dir)

.PHONY: clean

clean:
	-rm -r $(work_dir)
	-rm -r web/dist
