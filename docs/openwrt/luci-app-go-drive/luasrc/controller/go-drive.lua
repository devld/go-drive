-- Copyright 2024 OpenWrt
-- Licensed to the public under the Apache License 2.0.

module("luci.controller.go-drive", package.seeall)

function index()
	if not nixio.fs.access("/etc/config/go-drive") then
		return
	end

	local page
	
	page = entry({"admin", "services", "go-drive"}, template("go-drive"), _("Go-Drive"), 60)
	page.dependent = true
	page.acl_depends = { "luci-app-go-drive" }
	
	page = entry({"admin", "services", "go-drive", "status"}, call("act_status"))
	page.leaf = true
	
	page = entry({"admin", "services", "go-drive", "action"}, call("act_action"))
	page.leaf = true
end

function act_status()
	local sys = require "luci.sys"
	local util = require "luci.util"
	local result = {}
	
	local status = sys.call("pgrep go-drive >/dev/null") == 0
	
	result.running = status
	if status then
		result.pid = util.trim(sys.exec("pgrep go-drive"))
	end
	
	luci.http.prepare_content("application/json")
	luci.http.write_json(result)
end

function act_action()
	local action = luci.http.formvalue("action")
	local result = { success = false }
	
	if action == "start" then
		result.success = os.execute("/etc/init.d/go-drive start") == 0
	elseif action == "stop" then
		result.success = os.execute("/etc/init.d/go-drive stop") == 0
	elseif action == "restart" then
		result.success = os.execute("/etc/init.d/go-drive restart") == 0
	elseif action == "reload" then
		result.success = os.execute("/etc/init.d/go-drive reload") == 0
	end
	
	luci.http.prepare_content("application/json")
	luci.http.write_json(result)
end