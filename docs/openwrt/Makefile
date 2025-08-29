#
# Copyright (C) 2024 OpenWrt.org
#
# This is free software, licensed under the GNU General Public License v2.
# See /LICENSE for more information.
#

include $(TOPDIR)/rules.mk

PKG_NAME:=go-drive
PKG_VERSION:=1.0.0
PKG_RELEASE:=1

PKG_SOURCE_PROTO:=git
PKG_SOURCE_URL:=https://github.com/devld/go-drive.git
PKG_SOURCE_VERSION:=HEAD
PKG_SOURCE_SUBDIR:=$(PKG_NAME)-$(PKG_VERSION)
PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz

PKG_MAINTAINER:=Go-Drive Team
PKG_LICENSE:=MIT
PKG_LICENSE_FILES:=LICENSE

PKG_BUILD_DEPENDS:=golang/host node/host
PKG_BUILD_PARALLEL:=1
PKG_USE_MIPS16:=0

GO_PKG:=go-drive
GO_PKG_EXCLUDES:=vendor
GO_PKG_BUILD_PKG:=go-drive

include $(INCLUDE_DIR)/package.mk
include ../../lang/golang/golang-package.mk

define Package/go-drive
  SECTION:=net
  CATEGORY:=Network
  SUBMENU:=Cloud Manager
  TITLE:=A powerful cloud drive service
  URL:=https://github.com/devld/go-drive
  DEPENDS:=$(GO_ARCH_DEPENDS) +ca-bundle
endef

define Package/go-drive/description
  Go-Drive is a powerful cloud drive service that supports multiple storage 
  backends including local storage, OneDrive, Google Drive, and more.
  It provides a modern web interface and RESTful API for file management.
endef

define Package/go-drive/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/go-drive $(1)/usr/bin/

	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/etc/config/go-drive $(1)/etc/config/

	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/etc/init.d/go-drive $(1)/etc/init.d/

	$(INSTALL_DIR) $(1)/etc/uci-defaults
	$(INSTALL_BIN) ./files/etc/uci-defaults/go-drive $(1)/etc/uci-defaults/

	$(INSTALL_DIR) $(1)/usr/share/go-drive/web
	$(CP) $(PKG_BUILD_DIR)/web/dist/* $(1)/usr/share/go-drive/web/

	$(INSTALL_DIR) $(1)/usr/share/go-drive/lang
	$(CP) $(PKG_BUILD_DIR)/docs/lang/* $(1)/usr/share/go-drive/lang/

	$(INSTALL_DIR) $(1)/etc/go-drive
	$(INSTALL_CONF) $(PKG_BUILD_DIR)/docs/config.yml $(1)/etc/go-drive/config.yml.example
endef

define Package/luci-app-go-drive
  SECTION:=luci
  CATEGORY:=LuCI
  SUBMENU:=3. Applications
  TITLE:=LuCI Support for Go-Drive
  DEPENDS:=+go-drive +luci-base
endef

define Package/luci-app-go-drive/description
  LuCI web interface for Go-Drive cloud storage service.
  Allows configuration and management of Go-Drive through OpenWrt's web interface.
endef

define Package/luci-app-go-drive/install
	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/controller
	$(INSTALL_DATA) ./luci-app-go-drive/luasrc/controller/go-drive.lua $(1)/usr/lib/lua/luci/controller/

	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/model/cbi
	$(INSTALL_DATA) ./luci-app-go-drive/luasrc/model/cbi/go-drive.lua $(1)/usr/lib/lua/luci/model/cbi/

	$(INSTALL_DIR) $(1)/www/luci-static/resources/view
	$(INSTALL_DATA) ./luci-app-go-drive/htdocs/luci-static/resources/view/go-drive.js $(1)/www/luci-static/resources/view/

	$(INSTALL_DIR) $(1)/usr/share/luci/menu.d
	$(INSTALL_DATA) ./luci-app-go-drive/root/usr/share/luci/menu.d/luci-app-go-drive.json $(1)/usr/share/luci/menu.d/
endef

define Build/Compile
	cd $(PKG_BUILD_DIR)/web && npm install && npm run build
	$(call GoPackage/Build/Compile)
endef

$(eval $(call GoBinPackage,go-drive))
$(eval $(call BuildPackage,go-drive))
$(eval $(call BuildPackage,luci-app-go-drive))