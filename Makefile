
work_dir = build
target_name = go-drive_$(GOOS)_$(GOARCH)
build_dir = $(work_dir)/$(target_name)

all: $(build_dir)/$(target_name).tar.gz

$(build_dir)/$(target_name).tar.gz: $(build_dir)/go-drive $(build_dir)/web
	cd $(work_dir); tar acf $(target_name).tar.gz --owner=0 --group=0 $(target_name)

$(build_dir)/go-drive: $(build_dir)
	go build -o $(build_dir)

$(build_dir)/web: $(build_dir)
	cd web; npm install && npm run build
	cp -R web/dist $(build_dir)/web

$(build_dir): check-env
	mkdir -p $(build_dir)

.PHONY: clean check-env

check-env:
ifndef GOOS
	$(error GOOS is undefined)
endif
ifndef GOARCH
	$(error GOARCH is undefined)
endif

clean:
	-rm -r $(work_dir)
	-rm -r web/dist
