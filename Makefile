# Makefile for Alfred. Run 'make' to build locally, 'make install' to install binaries and other
# data, 'make package' to prepare a redistributable package.
# 
# User-defined build options.
# 
COMPILER = gc
PROGRAM  = alfred
VERSION  = 1.0.0
REPO     = github.com/Hearst-Digital/alfred

# No editing from here on!

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
	@echo -e "\033[1mInstalling '$(PROGRAM)' and data...\033[0m"

	@install -s -Dm 755 .tmp/$(PROGRAM) $(DESTDIR)/usr/bin/$(PROGRAM)

package:
	@echo -e "\033[1mBuilding package for '$(PROGRAM)'...\033[0m"

	@mkdir -p .tmp/package
	@make DESTDIR=.tmp/package install
	@tar -cJf $(PROGRAM)-$(VERSION).tar.xz -C .tmp/package .

uninstall:
	@echo -e "\033[1mUninstalling '$(PROGRAM)'...\033[0m"

	@rm -f $(DESTDIR)/usr/bin/$(PROGRAM)

clean:
	@echo -e "\033[1mCleaning '$(PROGRAM)'...\033[0m"

	@go clean
	@rm -Rf .tmp services.go
