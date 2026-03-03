import {
  EuiButton,
  EuiCallOut,
  EuiFieldNumber,
  EuiFieldText,
  EuiForm,
  EuiFormRow,
  EuiPanel,
  EuiSpacer,
  EuiSwitch,
  EuiTab,
  EuiTabs,
  EuiTextArea,
  EuiTitle,
} from '@elastic/eui';

import { useCallback, useEffect, useState } from 'react';

// Defaults match CLI flags (cmd/root.go). Used to prepopulate when key missing and to omit from JSON when value equals default.
const SETTINGS_DEFAULTS = {
  m3u_url: '',
  m3u_file_name: 'iptv.m3u',
  custom_endpoint: '',
  custom_id: '',
  port: 8080,
  advertised_port: 0,
  hostname: '',
  https: false,
  user: '',
  password: '',
  xtream_user: '',
  xtream_password: '',
  xtream_base_url: '',
  xtream_api_get: false,
  m3u_cache_expiration: 1,
  divide_by_res: false,
  xmltv_cache_ttl: '',
  xmltv_cache_max_entries: 100,
  debug_logging: false,
  cache_folder: '',
  use_xtream_advanced_parsing: false,
  ui_port: 8081,
};

// Categories for grouping. Fields not in any category go under "Other".
const SETTINGS_CATEGORIES = [
  { id: 'input', title: 'Input (M3U URL or Xtream)', description: 'Source is either an M3U URL (below) or Xtream credentials (Xtream section). Use one or the other.' },
  { id: 'serving', title: 'Serving' },
  { id: 'output', title: 'Output' },
  { id: 'xtream', title: 'Xtream (alternative input)', description: 'Use when your source is Xtream API instead of an M3U URL. Leave M3U URL empty when using Xtream.' },
  { id: 'cache', title: 'Cache & EPG' },
  { id: 'other', title: 'Other' },
];

// All settings keys with label, type, help (example text), and category.
const SETTINGS_FIELDS = [
  { key: 'm3u_url', label: 'M3U URL', type: 'text', category: 'input', help: 'Input type: M3U playlist URL. Use this or Xtream credentials, not both.', example: 'http://example.com/get.php?username=user&password=pass&type=m3u_plus&output=m3u8' },
  { key: 'm3u_file_name', label: 'M3U file name', type: 'text', category: 'output', help: 'Filename of the proxified playlist.', example: 'iptv.m3u' },
  { key: 'custom_endpoint', label: 'Custom endpoint', type: 'text', category: 'output', help: 'Path prefix for M3U in the URL.', example: 'api (yields …/api/iptv.m3u)' },
  { key: 'custom_id', label: 'Custom ID', type: 'text', category: 'output', help: 'Anti-collision path segment for track URLs.', example: '' },
  { key: 'port', label: 'Port', type: 'number', category: 'serving', help: 'Port the proxy listens on.', example: '8080' },
  { key: 'advertised_port', label: 'Advertised port', type: 'number', category: 'serving', help: 'Port in generated URLs (0 = use port); set when behind reverse proxy.', example: '0 or 443' },
  { key: 'hostname', label: 'Hostname', type: 'text', category: 'serving', help: 'Hostname or IP used in generated playlist/stream URLs.', example: 'localhost' },
  { key: 'https', label: 'HTTPS', type: 'boolean', category: 'serving', help: 'Use https in generated URLs.', example: '' },
  { key: 'user', label: 'User', type: 'text', category: 'serving', help: 'Proxy auth username (M3U and Xtream).', example: '' },
  { key: 'password', label: 'Password', type: 'text', category: 'serving', help: 'Proxy auth password (M3U and Xtream).', example: '' },
  { key: 'ui_port', label: 'UI port', type: 'number', category: 'serving', help: 'Port for configuration UI (0 = disabled).', example: '8081' },
  { key: 'xtream_user', label: 'Xtream user', type: 'text', category: 'xtream', help: 'Xtream provider username.', example: '' },
  { key: 'xtream_password', label: 'Xtream password', type: 'text', category: 'xtream', help: 'Xtream provider password.', example: '' },
  { key: 'xtream_base_url', label: 'Xtream base URL', type: 'text', category: 'xtream', help: 'Xtream provider base URL.', example: 'http://provider.tv:8080' },
  { key: 'xtream_api_get', label: 'Xtream API get', type: 'boolean', category: 'xtream', help: 'Serve get.php from Xtream API instead of provider endpoint.', example: '' },
  { key: 'm3u_cache_expiration', label: 'M3U cache expiration (hours)', type: 'number', category: 'cache', help: 'M3U cache TTL in hours (Xtream-generated M3U).', example: '1' },
  { key: 'divide_by_res', label: 'Divide by resolution', type: 'boolean', category: 'output', help: 'Add resolution suffix to groups (FHD/HD/SD) and strip from names.', example: '' },
  { key: 'xmltv_cache_ttl', label: 'XMLTV cache TTL', type: 'text', category: 'cache', help: 'XMLTV (EPG) cache TTL.', example: '1h or 30m' },
  { key: 'xmltv_cache_max_entries', label: 'XMLTV cache max entries', type: 'number', category: 'cache', help: 'Maximum number of cached XMLTV responses.', example: '100' },
  { key: 'debug_logging', label: 'Debug logging', type: 'boolean', category: 'other', help: 'Enable verbose debug logging.', example: '' },
  { key: 'cache_folder', label: 'Cache folder', type: 'text', category: 'other', help: 'Folder to save provider/client responses (debug).', example: '' },
  { key: 'use_xtream_advanced_parsing', label: 'Use Xtream advanced parsing', type: 'boolean', category: 'xtream', help: 'Use alternate Xtream response parsing for some providers.', example: '' },
];

function getDefault(key) {
  const d = SETTINGS_DEFAULTS[key];
  if (d === undefined || d === null) return '';
  if (typeof d === 'boolean') return d;
  return String(d);
}

function valueEqualsDefault(key, value, type) {
  const def = getDefault(key);
  if (type === 'boolean') return !!value === !!def;
  if (type === 'number') return (parseInt(value, 10) || 0) === (parseInt(def, 10) || 0);
  return (value || '') === (def || '');
}

export function SettingsPage() {
  const [selectedTabId, setSelectedTabId] = useState('options');
  const [settings, setSettings] = useState(null);
  const [formValues, setFormValues] = useState({});
  const [inSettings, setInSettings] = useState({}); // key -> true if value came from settings.json
  const [raw, setRaw] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState(null);
  const [error, setError] = useState(null);

  const fetchSettings = useCallback(() => {
    setLoading(true);
    setError(null);
    fetch('/api/settings')
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        return res.json();
      })
      .then((data) => {
        // API returns { file: actual settings.json, effective: file merged with flag/env }
        const file = data.file || {};
        const effective = data.effective || {};
        setSettings({ file, effective });
        setRaw(JSON.stringify(file, null, 2));
        const vals = {};
        const inSet = {};
        SETTINGS_FIELDS.forEach((f) => {
          const fromEffective = effective[f.key];
          const inFile = file[f.key] !== undefined && file[f.key] !== null && (f.type !== 'text' || file[f.key] !== '');
          if (fromEffective !== undefined && fromEffective !== null && (f.type !== 'text' || fromEffective !== '')) {
            if (f.type === 'boolean') vals[f.key] = !!fromEffective;
            else vals[f.key] = String(fromEffective);
          } else {
            vals[f.key] = f.type === 'boolean' ? getDefault(f.key) : (getDefault(f.key) || '');
          }
          inSet[f.key] = inFile;
        });
        setFormValues(vals);
        setInSettings(inSet);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => fetchSettings(), [fetchSettings]);

  const updateForm = (key, value) => {
    setFormValues((prev) => ({ ...prev, [key]: value }));
  };

  const saveFromForm = () => {
    const base = settings?.effective ? { ...settings.effective } : {};
    const next = { ...base };
    SETTINGS_FIELDS.forEach((f) => {
      const v = formValues[f.key];
      if (valueEqualsDefault(f.key, v, f.type)) {
        delete next[f.key];
      } else if (f.type === 'text' && (v === undefined || v === null || String(v).trim() === '')) {
        delete next[f.key];
      } else {
        if (f.type === 'boolean') next[f.key] = !!v;
        else if (f.type === 'number') {
          const n = parseInt(v, 10);
          if (!isNaN(n)) next[f.key] = n;
        } else next[f.key] = v;
      }
    });

    setSaving(true);
    setMessage(null);
    setError(null);
    fetch('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(next),
    })
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        setMessage('Saved. Changes apply immediately.');
        fetchSettings();
      })
      .catch((e) => setMessage('Error: ' + e.message))
      .finally(() => setSaving(false));
  };

  const saveRaw = () => {
    let parsed;
    try {
      parsed = JSON.parse(raw);
    } catch (e) {
      setMessage('Invalid JSON: ' + e.message);
      return;
    }
    setSaving(true);
    setMessage(null);
    setError(null);
    fetch('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(parsed),
    })
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        setMessage('Saved. Changes apply immediately.');
        fetchSettings();
      })
      .catch((e) => setMessage('Error: ' + e.message))
      .finally(() => setSaving(false));
  };

  const jsonInvalid = raw.trim() !== '' && (() => {
    try {
      JSON.parse(raw);
      return false;
    } catch {
      return true;
    }
  })();

  const fieldsByCategory = SETTINGS_CATEGORIES.map((cat) => ({
    ...cat,
    fields: SETTINGS_FIELDS.filter((f) => (f.category || 'other') === cat.id),
  })).filter((c) => c.fields.length > 0);

  return (
    <EuiPanel paddingSize="l">
      <EuiTitle size="m">
        <h2>Settings (settings.json)</h2>
      </EuiTitle>
      <EuiSpacer size="m" />
      <EuiTabs>
        <EuiTab onClick={() => setSelectedTabId('options')} isSelected={selectedTabId === 'options'}>
          Options
        </EuiTab>
        <EuiTab onClick={() => setSelectedTabId('raw')} isSelected={selectedTabId === 'raw'}>
          Raw JSON
        </EuiTab>
      </EuiTabs>
      <EuiSpacer size="m" />

      {error && (
        <>
          <EuiCallOut title="Error loading settings" color="danger" iconType="alert">
            <p>{error}</p>
          </EuiCallOut>
          <EuiSpacer size="m" />
        </>
      )}
      {message && (
        <>
          <EuiCallOut
            title={message.includes('Error') ? 'Error' : 'Saved'}
            color={message.includes('Error') ? 'danger' : 'success'}
            iconType={message.includes('Error') ? 'alert' : 'check'}
          >
            <p>{message}</p>
          </EuiCallOut>
          <EuiSpacer size="m" />
        </>
      )}

      {selectedTabId === 'options' && (
        <>
          <p className="euiTextColor--subdued">
            Values from settings.json are shown; missing keys show the default (grayed). Save only writes values that differ from the default.
          </p>
          <EuiSpacer size="m" />
          <EuiForm component="form" onSubmit={(e) => { e.preventDefault(); saveFromForm(); }}>
            {fieldsByCategory.map(({ id, title, description, fields }) => (
              <div key={id}>
                <EuiTitle size="xs"><h3>{title}</h3></EuiTitle>
                {description && (
                  <>
                    <p className="euiTextColor--subdued" style={{ marginTop: 4, marginBottom: 0 }}>{description}</p>
                    <EuiSpacer size="s" />
                  </>
                )}
                <EuiSpacer size="s" />
                {fields.map((f) => {
                  const helpText = f.example ? `${f.help} Example: ${f.example}` : f.help;
                  const isFromSettings = inSettings[f.key];
                  return (
                    <EuiFormRow key={f.key} label={f.label} helpText={helpText} fullWidth>
                      {f.type === 'boolean' ? (
                        <EuiSwitch
                          label=""
                          checked={!!formValues[f.key]}
                          onChange={(e) => updateForm(f.key, e.target.checked)}
                          disabled={loading}
                        />
                      ) : f.type === 'number' ? (
                        <EuiFieldNumber
                          value={String(formValues[f.key] ?? '')}
                          onChange={(e) => updateForm(f.key, e.target.value)}
                          disabled={loading}
                          min={0}
                          style={{ maxWidth: 140 }}
                        />
                      ) : (
                        <EuiFieldText
                          value={formValues[f.key] || ''}
                          onChange={(e) => updateForm(f.key, e.target.value)}
                          disabled={loading}
                          fullWidth
                          placeholder={!isFromSettings ? (getDefault(f.key) || '') : undefined}
                          style={!isFromSettings && (formValues[f.key] || '') === (getDefault(f.key) || '') ? { color: '#69707d' } : undefined}
                        />
                      )}
                    </EuiFormRow>
                  );
                })}
                <EuiSpacer size="l" />
              </div>
            ))}
            <EuiSpacer size="m" />
            <EuiButton type="submit" fill isLoading={saving} isDisabled={loading}>
              Save settings
            </EuiButton>
          </EuiForm>
        </>
      )}

      {selectedTabId === 'raw' && (
        <>
          <p className="euiTextColor--subdued">
            Edit the full settings.json. Include replacements here if you edit manually.
          </p>
          <EuiSpacer size="m" />
          <EuiFormRow label="settings.json" fullWidth>
            <EuiTextArea
              value={raw}
              onChange={(e) => setRaw(e.target.value)}
              fullWidth
              rows={20}
              isInvalid={jsonInvalid}
              isLoading={loading}
              readOnly={loading}
            />
          </EuiFormRow>
          <EuiSpacer size="m" />
          <EuiButton onClick={saveRaw} fill isLoading={saving} isDisabled={loading || jsonInvalid}>
            Save settings
          </EuiButton>
        </>
      )}
    </EuiPanel>
  );
}
