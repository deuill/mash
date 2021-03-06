# Makefile for Mash. Run 'make' to build locally, 'make install' to install binaries and other
# data, 'make package' to prepare a redistributable package.
#
# User-defined build options.
#
COMPILER = gc
PROGRAM  = mash
REPO     = github.com/deuill/mash

# No editing from here on!

VERSION  = $(shell git describe --tags | cut -c3-)
SERVICES = $(shell find service/* -maxdepth 1 -type d)

.PHONY: $(PROGRAM)
all: $(PROGRAM)

$(PROGRAM): depend
	@echo -e "\033[1mBuilding '$(PROGRAM)'...\033[0m"

	@mkdir -p .tmp
	@go build -compiler $(COMPILER) -o .tmp/$(PROGRAM)

depend:
	$(shell echo "package main"  > services.go)
	$(foreach srv, $(SERVICES), $(shell echo "import _ \"$(REPO)/$(srv)\""  >> services.go))

install:
	@echo -e "\033[1mInstalling '$(PROGRAM)'...\033[0m"

ifneq ($(wildcard /etc/systemd),)
	@install -Dm 644 dist/init/systemd/$(PROGRAM).service $(DESTDIR)/usr/lib/systemd/system/$(PROGRAM).service
else
	@install -Dm 755 dist/init/systemv/$(PROGRAM) $(DESTDIR)/etc/rc.d/init.d/$(PROGRAM)
	@install -Dm 644 dist/init/systemv/default $(DESTDIR)/etc/default/$(PROGRAM)
endif

	@install -dm 0750 $(DESTDIR)/etc/$(PROGRAM)
	@install -m 0640 dist/conf/* $(DESTDIR)/etc/$(PROGRAM)

	@install -Dsm 0755 .tmp/$(PROGRAM) $(DESTDIR)/usr/bin/$(PROGRAM)

package:
	@echo -e "\033[1mBuilding package for '$(PROGRAM)'...\033[0m"

	@mkdir -p .tmp/package
	@make DESTDIR=.tmp/package install
	@fakeroot -- tar -cJf $(PROGRAM)-$(VERSION).tar.xz -C .tmp/package .

rpm:
	@echo -e "\033[1mBuilding RPM package for '$(PROGRAM)'...\033[0m"

	@mkdir -p .tmp/package
	@make DESTDIR=.tmp/package install
	@fpm -s dir -t rpm -n $(PROGRAM) -v $(VERSION) \
	     --config-files etc/$(PROGRAM) \
	     --after-install dist/pkg/post-install \
	     --after-remove dist/pkg/post-remove \
	     -C .tmp/package .

uninstall:
	@echo -e "\033[1mUninstalling '$(PROGRAM)'...\033[0m"

ifneq ($(wildcard /etc/systemd),)
	@rm -f $(DESTDIR)/usr/lib/systemd/system/$(PROGRAM).service
else
	@rm -f $(DESTDIR)/etc/rc.d/init.d/$(PROGRAM) $(DESTDIR)/etc/default/$(PROGRAM)
endif

	@rm -f $(DESTDIR)/usr/bin/$(PROGRAM)
	@mv -f $(DESTDIR)/etc/$(PROGRAM) $(DESTDIR)/etc/$(PROGRAM).save

	@echo -e "\033[1mConfiguration files moved to '$(DESTDIR)/etc/$(PROGRAM).save'.\033[0m"

clean:
	@echo -e "\033[1mCleaning '$(PROGRAM)'...\033[0m"

	@go clean
	@rm -Rf .tmp services.go
