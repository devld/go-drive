
work_dir = build
target_name = go-drive_$(GOOS)_$(GOARCH)
build_dir = $(work_dir)/$(target_name)

all: $(build_dir)/$(target_name).tar.gz
zip: $(build_dir)/$(target_name).zip

# tar.gz
$(build_dir)/$(target_name).tar.gz: $(build_dir)/go-drive $(build_dir)/web $(build_dir)/lang
	cd $(work_dir); tar acf $(target_name).tar.gz --owner=0 --group=0 $(target_name)

# zip for windows
$(build_dir)/$(target_name).zip: $(build_dir)/go-drive $(build_dir)/web $(build_dir)/lang
	cd $(work_dir); zip -q -r $(target_name).zip $(target_name)

$(build_dir)/go-drive: $(build_dir)
	go build -o $(build_dir) -ldflags \
		"-X go-drive/common.version=${BUILD_VERSION} -X go-drive/common.hash=$(shell git rev-parse HEAD) -X go-drive/common.build=$(shell date +'%Y%m%d')"

$(build_dir)/web: $(build_dir) web/dist
	cp -R web/dist $(build_dir)/web

$(build_dir)/lang: $(build_dir)
	cp -R docs/lang $(build_dir)/

web/dist:
	cd web; npm install && npm run build

$(build_dir): check-env
	mkdir -p $(build_dir)

.PHONY: clean check-env

clean:
	-rm -r $(work_dir)
	-rm -r web/dist

check-env:
ifndef GOOS
	$(error GOOS is undefined)
endif
ifndef GOARCH
	$(error GOARCH is undefined)
endif
