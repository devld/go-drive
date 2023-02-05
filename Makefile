
work_dir = build
target_name ?= go-drive_build
build_dir = $(work_dir)/$(target_name)

all: $(build_dir)/$(target_name).tar.gz
zip: $(build_dir)/$(target_name).zip

# tar.gz
$(build_dir)/$(target_name).tar.gz: $(build_dir)/$(target_name)
	cd $(work_dir); tar acf $(target_name).tar.gz --owner=0 --group=0 $(target_name)

# zip for windows
$(build_dir)/$(target_name).zip: $(build_dir)/$(target_name)
	cd $(work_dir); zip -q -r $(target_name).zip $(target_name)

$(build_dir)/$(target_name): $(build_dir)/go-drive $(build_dir)/web $(build_dir)/lang $(build_dir)/config.yml

$(build_dir)/go-drive: $(build_dir)
	CGO_CFLAGS="-Wno-return-local-addr" \
	go build -o $(build_dir) -ldflags \
		"-w -s \
		-X 'go-drive/common.Version=${BUILD_VERSION}' \
		-X 'go-drive/common.RevHash=$(shell git rev-parse HEAD)' \
		-X 'go-drive/common.BuildAt=$(shell date -R)'"

$(build_dir)/web: $(build_dir) web/dist
	cp -R web/dist $(build_dir)/web

$(build_dir)/lang: $(build_dir)
	cp -R docs/lang $(build_dir)/

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
