package server

const uiHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>IPTV-Proxy configuration</title>
  <style>
    * { box-sizing: border-box; }
    body { font-family: system-ui, sans-serif; margin: 0; padding: 1rem; background: #1a1a2e; color: #eee; }
    h1 { margin-top: 0; }
    .tabs { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
    .tabs button { padding: 0.5rem 1rem; cursor: pointer; background: #16213e; border: 1px solid #0f3460; color: #eee; border-radius: 4px; }
    .tabs button.active { background: #0f3460; }
    .tabs button:hover { background: #1a1a2e; }
    .panel { display: none; }
    .panel.active { display: block; }
    table { width: 100%; border-collapse: collapse; margin-top: 0.5rem; }
    th, td { padding: 0.4rem 0.6rem; text-align: left; border: 1px solid #333; }
    th { background: #16213e; }
    tr:nth-child(even) { background: #16213e33; }
    input, button { padding: 0.4rem; margin: 0.2rem; background: #0f3460; border: 1px solid #333; color: #eee; border-radius: 4px; }
    button { cursor: pointer; }
    button:hover { background: #1a1a2e; }
    .rule-row { display: flex; gap: 0.5rem; align-items: center; margin-bottom: 0.5rem; }
    .rule-row input { flex: 1; min-width: 0; }
    .add-rule { margin-top: 1rem; }
    .section { margin-bottom: 2rem; }
    .section h3 { margin-bottom: 0.5rem; }
    .error { color: #e74c3c; }
    .info { color: #3498db; font-size: 0.9rem; margin-bottom: 1rem; }
  </style>
</head>
<body>
  <h1>IPTV-Proxy configuration</h1>
  <p class="info">Manage groups, channels, and replacement rules. Changes to replacements.json take effect after restarting the proxy.</p>
  <div class="tabs">
    <button type="button" data-tab="groups">Groups</button>
    <button type="button" data-tab="channels">Channels</button>
    <button type="button" data-tab="replacements">Replacements</button>
  </div>
  <div id="groups" class="panel">
    <div class="section">
      <h3>Group titles (from playlist)</h3>
      <p class="info">Add a rule from a value: use "Add to groups-replacements" in the Replacements tab.</p>
      <table><thead><tr><th>#</th><th>Group title</th></tr></thead><tbody id="groups-body"></tbody></table>
    </div>
  </div>
  <div id="channels" class="panel">
    <div class="section">
      <h3>Channels (from playlist)</h3>
      <p class="info">Add a rule from a name: use "Add to names-replacements" in the Replacements tab.</p>
      <table><thead><tr><th>Name</th><th>Group</th><th>tvg-id</th><th>tvg-name</th><th>tvg-logo</th></tr></thead><tbody id="channels-body"></tbody></table>
    </div>
  </div>
  <div id="replacements" class="panel">
    <div class="section">
      <h3>Global replacements</h3>
      <p class="info">Applied to both channel names and group titles.</p>
      <div id="global-rules"></div>
      <button type="button" class="add-rule" data-section="global">+ Add rule</button>
    </div>
    <div class="section">
      <h3>Names replacements</h3>
      <p class="info">Applied only to channel names.</p>
      <div id="names-rules"></div>
      <button type="button" class="add-rule" data-section="names">+ Add rule</button>
    </div>
    <div class="section">
      <h3>Groups replacements</h3>
      <p class="info">Applied only to group-title values.</p>
      <div id="groups-rules"></div>
      <button type="button" class="add-rule" data-section="groups">+ Add rule</button>
    </div>
    <div><button type="button" id="save-replacements">Save replacements.json</button><span id="save-status"></span></div>
  </div>
  <script>
    const api = (path, opts = {}) => fetch(path, { headers: { 'Content-Type': 'application/json', ...opts.headers }, ...opts }).then(r => { if (!r.ok) throw new Error(r.statusText); return r.json().catch(() => ({})); });
    function showTab(id) {
      document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
      document.querySelectorAll('.tabs button').forEach(b => b.classList.remove('active'));
      document.getElementById(id).classList.add('active');
      document.querySelector('.tabs button[data-tab="' + id + '"]').classList.add('active');
    }
    document.querySelectorAll('.tabs button').forEach(b => b.addEventListener('click', () => showTab(b.dataset.tab)));
    function loadGroups() {
      api('/api/groups').then(arr => {
        const tbody = document.getElementById('groups-body');
        tbody.innerHTML = arr.map((g, i) => '<tr><td>' + (i+1) + '</td><td>' + escapeHtml(g) + '</td></tr>').join('') || '<tr><td colspan="2">No groups (no M3U loaded or playlist empty)</td></tr>';
      }).catch(e => { document.getElementById('groups-body').innerHTML = '<tr><td colspan="2" class="error">' + escapeHtml(e.message) + '</td></tr>'; });
    }
    function loadChannels() {
      api('/api/channels').then(arr => {
        const tbody = document.getElementById('channels-body');
        tbody.innerHTML = arr.map(c => '<tr><td>' + escapeHtml(c.name) + '</td><td>' + escapeHtml(c.group) + '</td><td>' + escapeHtml(c.tvg_id) + '</td><td>' + escapeHtml(c.tvg_name) + '</td><td>' + escapeHtml(c.tvg_logo) + '</td></tr>').join('') || '<tr><td colspan="5">No channels</td></tr>';
      }).catch(e => { document.getElementById('channels-body').innerHTML = '<tr><td colspan="5" class="error">' + escapeHtml(e.message) + '</td></tr>'; });
    }
    function escapeHtml(s) {
      if (s == null) return '';
      const div = document.createElement('div');
      div.textContent = s;
      return div.innerHTML;
    }
    let replacements = { 'global-replacements': [], 'names-replacements': [], 'groups-replacements': [] };
    function renderRule(section, index, r) {
      const div = document.createElement('div');
      div.className = 'rule-row';
      div.innerHTML = '<input placeholder="regex replace" value="' + escapeHtml(r.replace) + '">' +
        '<input placeholder="with" value="' + escapeHtml(r.with) + '">' +
        '<button type="button" data-action="remove">Remove</button>';
      const key = section + '-replacements';
      const inp = div.querySelectorAll('input');
      div.querySelector('[data-action="remove"]').addEventListener('click', () => {
        replacements[key].splice(index, 1);
        loadReplacementsUI();
      });
      inp[0].addEventListener('change', () => { replacements[key][index].replace = inp[0].value; });
      inp[1].addEventListener('change', () => { replacements[key][index].with = inp[1].value; });
      return div;
    }
    function loadReplacementsUI() {
      ['global', 'names', 'groups'].forEach(section => {
        const key = section + '-replacements';
        const container = document.getElementById(section + '-rules');
        container.innerHTML = '';
        (replacements[key] || []).forEach((r, i) => container.appendChild(renderRule(section, i, r)));
      });
    }
    function loadReplacements() {
      api('/api/replacements').then(data => {
        replacements = {
          'global-replacements': data['global-replacements'] || [],
          'names-replacements': data['names-replacements'] || [],
          'groups-replacements': data['groups-replacements'] || []
        };
        loadReplacementsUI();
      }).catch(e => { document.getElementById('save-status').textContent = 'Load error: ' + e.message; document.getElementById('save-status').className = 'error'; });
    }
    document.querySelectorAll('.add-rule').forEach(b => b.addEventListener('click', () => {
      const section = b.dataset.section;
      const key = section + '-replacements';
      if (!replacements[key]) replacements[key] = [];
      replacements[key].push({ replace: '', with: '' });
      loadReplacementsUI();
    }));
    document.getElementById('save-replacements').addEventListener('click', () => {
      const status = document.getElementById('save-status');
      status.textContent = ' Saving...';
      status.className = '';
      fetch('/api/replacements', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(replacements) })
        .then(r => { if (!r.ok) throw new Error(r.statusText); status.textContent = ' Saved.'; status.className = 'info'; })
        .catch(e => { status.textContent = ' Error: ' + e.message; status.className = 'error'; });
    });
    loadGroups();
    loadChannels();
    loadReplacements();
    showTab('groups');
  </script>
</body>
</html>
`
