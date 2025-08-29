-- Copyright 2024 OpenWrt
-- Licensed to the public under the Apache License 2.0.

local m, s, o
local sys = require "luci.sys"

m = Map("go-drive", translate("Go-Drive"), translate("Go-Drive is a powerful cloud drive service that supports multiple storage backends."))

-- Service Status
s = m:section(TypedSection, "go-drive", translate("Service Status"))
s.anonymous = true

local status = sys.call("pgrep go-drive >/dev/null") == 0
local status_text = status and translate("RUNNING") or translate("NOT RUNNING")
local status_color = status and "green" or "red"

s:option(DummyValue, "status", translate("Status")).value = 
	string.format('<span style="color:%s"><strong>%s</strong></span>', status_color, status_text)

-- Basic Settings
s = m:section(NamedSection, "config", "go-drive", translate("Basic Settings"))

o = s:option(Flag, "enabled", translate("Enable"))
o.default = "0"

o = s:option(Value, "listen", translate("Listen Address"))
o.default = ":8089"
o.placeholder = ":8089"

o = s:option(Value, "data_dir", translate("Data Directory"))
o.default = "/opt/go-drive"
o.placeholder = "/opt/go-drive"

o = s:option(Value, "web_dir", translate("Web Directory"))
o.default = "/usr/share/go-drive/web"
o.readonly = true

o = s:option(Value, "lang_dir", translate("Language Directory"))
o.default = "/usr/share/go-drive/lang"
o.readonly = true

o = s:option(ListValue, "default_lang", translate("Default Language"))
o:value("en-US", translate("English (US)"))
o:value("zh-CN", translate("Simplified Chinese"))
o:value("zh-TW", translate("Traditional Chinese"))
o.default = "en-US"

o = s:option(Value, "temp_dir", translate("Temporary Directory"))
o.placeholder = "/tmp/go-drive"

o = s:option(Value, "max_concurrent_task", translate("Max Concurrent Tasks"))
o.datatype = "uinteger"
o.default = "100"
o.placeholder = "100"

o = s:option(Flag, "free_fs", translate("Free Filesystem"))
o.default = "0"
o.description = translate("Allow absolute paths for Local Drive. WARNING: Admin users can access all system files!")

o = s:option(Value, "api_path", translate("API Path"))
o.placeholder = "/api"
o.description = translate("API path for reverse proxy setups")

o = s:option(Value, "web_path", translate("Web Path"))
o.placeholder = "/"
o.description = translate("Web path for reverse proxy setups")

-- Database Settings
s = m:section(NamedSection, "db", "database", translate("Database Settings"))

o = s:option(ListValue, "type", translate("Database Type"))
o:value("sqlite", "SQLite")
o:value("mysql", "MySQL")
o.default = "sqlite"

o = s:option(Value, "name", translate("Database Name"))
o.default = "data.db"

o = s:option(Value, "host", translate("Database Host"))
o:depends("type", "mysql")
o.placeholder = "127.0.0.1"

o = s:option(Value, "port", translate("Database Port"))
o:depends("type", "mysql")
o.datatype = "port"
o.placeholder = "3306"

o = s:option(Value, "user", translate("Database User"))
o:depends("type", "mysql")

o = s:option(Value, "password", translate("Database Password"))
o:depends("type", "mysql")
o.password = true

-- Thumbnail Settings
s = m:section(NamedSection, "thumbnail", "thumbnail", translate("Thumbnail Settings"))

o = s:option(Value, "ttl", translate("Cache TTL"))
o.default = "720h"
o.placeholder = "720h"
o.description = translate("Thumbnail cache validity period")

o = s:option(Value, "concurrent", translate("Concurrent Tasks"))
o.datatype = "uinteger"
o.default = "4"
o.placeholder = "4"
o.description = translate("Concurrent thumbnail generation tasks")

-- Authentication Settings
s = m:section(NamedSection, "auth", "auth", translate("Authentication Settings"))

o = s:option(Value, "validity", translate("Session Validity"))
o.default = "2h"
o.placeholder = "2h"
o.description = translate("User session validity period")

o = s:option(Flag, "auto_refresh", translate("Auto Refresh"))
o.default = "1"
o.description = translate("Auto refresh token when user is active")

-- Search Settings
s = m:section(NamedSection, "search", "search", translate("Search Settings"))

o = s:option(Flag, "enabled", translate("Enable Search"))
o.default = "0"

o = s:option(ListValue, "type", translate("Search Type"))
o:value("sqlite", "SQLite")
o:depends("enabled", "1")
o.default = "sqlite"

-- WebDAV Settings
s = m:section(NamedSection, "webdav", "webdav", translate("WebDAV Settings"))

o = s:option(Flag, "enabled", translate("Enable WebDAV"))
o.default = "0"

o = s:option(Value, "prefix", translate("WebDAV Prefix"))
o:depends("enabled", "1")
o.default = "/dav"
o.placeholder = "/dav"

o = s:option(Flag, "allow_anonymous", translate("Allow Anonymous"))
o:depends("enabled", "1")
o.default = "0"
o.description = translate("Allow anonymous WebDAV access")

o = s:option(Value, "max_cache_items", translate("Max Cache Items"))
o:depends("enabled", "1")
o.datatype = "uinteger"
o.default = "1000"
o.placeholder = "1000"

return m