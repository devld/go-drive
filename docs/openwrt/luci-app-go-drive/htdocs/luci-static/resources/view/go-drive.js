'use strict';
'require view';
'require form';
'require fs';
'require uci';
'require ui';
'require rpc';
'require poll';

var callServiceList = rpc.declare({
	object: 'service',
	method: 'list',
	params: ['name'],
	expect: { '': {} }
});

function getServiceStatus() {
	return L.resolveDefault(callServiceList('go-drive'), {}).then(function(res) {
		var isRunning = false;
		try {
			isRunning = res['go-drive']['instances']['instance1']['running'];
		} catch (e) { }
		return isRunning;
	});
}

function renderStatus(isRunning) {
	var spanTemp = '<em><span style="color:%s"><strong>%s %s</strong></span></em>';
	var renderHTML;
	if (isRunning) {
		renderHTML = String.format(spanTemp, 'green', _('Go-Drive'), _('RUNNING'));
	} else {
		renderHTML = String.format(spanTemp, 'red', _('Go-Drive'), _('NOT RUNNING'));
	}
	return renderHTML;
}

return view.extend({
	load: function() {
		return Promise.all([
			uci.load('go-drive')
		]);
	},

	render: function(data) {
		var m, s, o;

		m = new form.Map('go-drive', _('Go-Drive'), _('Go-Drive is a powerful cloud drive service that supports multiple storage backends.'));

		s = m.section(form.NamedSection, 'config', 'go-drive');

		o = s.option(form.DummyValue, '_status', _('Service Status'));
		o.rawhtml = true;
		o.render = function(option_index, section_id, in_table) {
			return E('div', { class: 'cbi-value' }, [
				E('label', { class: 'cbi-value-title' }, _('Service Status')),
				E('div', { class: 'cbi-value-field', id: 'service_status' }, E('em', _('Collecting data...')))
			]);
		};

		o = s.option(form.Flag, 'enabled', _('Enable'));
		o.default = '0';

		o = s.option(form.Value, 'listen', _('Listen Address'));
		o.default = ':8089';
		o.placeholder = ':8089';

		o = s.option(form.Value, 'data_dir', _('Data Directory'));
		o.default = '/opt/go-drive';
		o.placeholder = '/opt/go-drive';

		o = s.option(form.Value, 'web_dir', _('Web Directory'));
		o.default = '/usr/share/go-drive/web';
		o.readonly = true;

		o = s.option(form.Value, 'lang_dir', _('Language Directory'));
		o.default = '/usr/share/go-drive/lang';
		o.readonly = true;

		o = s.option(form.ListValue, 'default_lang', _('Default Language'));
		o.value('en-US', _('English (US)'));
		o.value('zh-CN', _('简体中文'));
		o.value('zh-TW', _('繁體中文'));
		o.default = 'en-US';

		o = s.option(form.Value, 'temp_dir', _('Temporary Directory'));
		o.placeholder = '/tmp/go-drive';

		o = s.option(form.Value, 'max_concurrent_task', _('Max Concurrent Tasks'));
		o.datatype = 'uinteger';
		o.default = '100';
		o.placeholder = '100';

		o = s.option(form.Flag, 'free_fs', _('Free Filesystem'));
		o.default = '0';
		o.description = _('Allow absolute paths for Local Drive. WARNING: Admin users can access all system files!');

		o = s.option(form.Value, 'api_path', _('API Path'));
		o.placeholder = '/api';
		o.description = _('API path for reverse proxy setups');

		o = s.option(form.Value, 'web_path', _('Web Path'));
		o.placeholder = '/';
		o.description = _('Web path for reverse proxy setups');

		// Database section
		s = m.section(form.NamedSection, 'db', 'database', _('Database Settings'));

		o = s.option(form.ListValue, 'type', _('Database Type'));
		o.value('sqlite', 'SQLite');
		o.value('mysql', 'MySQL');
		o.default = 'sqlite';

		o = s.option(form.Value, 'name', _('Database Name'));
		o.default = 'data.db';
		o.depends('type', 'sqlite');
		o.depends('type', 'mysql');

		o = s.option(form.Value, 'host', _('Database Host'));
		o.depends('type', 'mysql');
		o.placeholder = '127.0.0.1';

		o = s.option(form.Value, 'port', _('Database Port'));
		o.depends('type', 'mysql');
		o.datatype = 'port';
		o.placeholder = '3306';

		o = s.option(form.Value, 'user', _('Database User'));
		o.depends('type', 'mysql');

		o = s.option(form.Value, 'password', _('Database Password'));
		o.depends('type', 'mysql');
		o.password = true;

		// Thumbnail section
		s = m.section(form.NamedSection, 'thumbnail', 'thumbnail', _('Thumbnail Settings'));

		o = s.option(form.Value, 'ttl', _('Cache TTL'));
		o.default = '720h';
		o.placeholder = '720h';
		o.description = _('Thumbnail cache validity period');

		o = s.option(form.Value, 'concurrent', _('Concurrent Tasks'));
		o.datatype = 'uinteger';
		o.default = '4';
		o.placeholder = '4';
		o.description = _('Concurrent thumbnail generation tasks');

		// Auth section
		s = m.section(form.NamedSection, 'auth', 'auth', _('Authentication Settings'));

		o = s.option(form.Value, 'validity', _('Session Validity'));
		o.default = '2h';
		o.placeholder = '2h';
		o.description = _('User session validity period');

		o = s.option(form.Flag, 'auto_refresh', _('Auto Refresh'));
		o.default = '1';
		o.description = _('Auto refresh token when user is active');

		// Search section
		s = m.section(form.NamedSection, 'search', 'search', _('Search Settings'));

		o = s.option(form.Flag, 'enabled', _('Enable Search'));
		o.default = '0';

		o = s.option(form.ListValue, 'type', _('Search Type'));
		o.value('sqlite', 'SQLite');
		o.depends('enabled', '1');
		o.default = 'sqlite';

		// WebDAV section
		s = m.section(form.NamedSection, 'webdav', 'webdav', _('WebDAV Settings'));

		o = s.option(form.Flag, 'enabled', _('Enable WebDAV'));
		o.default = '0';

		o = s.option(form.Value, 'prefix', _('WebDAV Prefix'));
		o.depends('enabled', '1');
		o.default = '/dav';
		o.placeholder = '/dav';

		o = s.option(form.Flag, 'allow_anonymous', _('Allow Anonymous'));
		o.depends('enabled', '1');
		o.default = '0';
		o.description = _('Allow anonymous WebDAV access');

		o = s.option(form.Value, 'max_cache_items', _('Max Cache Items'));
		o.depends('enabled', '1');
		o.datatype = 'uinteger';
		o.default = '1000';
		o.placeholder = '1000';

		return m.render().then(function(mapEl) {
			poll.add(function() {
				return getServiceStatus().then(function(res) {
					var view = document.getElementById('service_status');
					if (view) {
						view.innerHTML = renderStatus(res);
					}
				});
			}, 5);

			return mapEl;
		});
	},

	handleSave: null,
	handleSaveApply: null,
	handleReset: null
});