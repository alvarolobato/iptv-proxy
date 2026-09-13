import {
  EuiBadge,
  EuiBasicTable,
  EuiButton,
  EuiButtonEmpty,
  EuiButtonIcon,
  EuiCallOut,
  EuiConfirmModal,
  EuiContextMenuItem,
  EuiContextMenuPanel,
  EuiFieldPassword,
  EuiFieldSearch,
  EuiFieldText,
  EuiFilterButton,
  EuiFilterGroup,
  EuiFlexGroup,
  EuiFlexItem,
  EuiFormRow,
  EuiIcon,
  EuiModal,
  EuiModalBody,
  EuiModalFooter,
  EuiModalHeader,
  EuiModalHeaderTitle,
  EuiPageTemplate,
  EuiPanel,
  EuiPopover,
  EuiScreenReaderOnly,
  EuiSelect,
  EuiSpacer,
  EuiSwitch,
  EuiTab,
  EuiTabs,
  EuiTitle,
  EuiToolTip,
} from '@elastic/eui';

import {
  Outlet,
  RouterProvider,
  createBrowserRouter,
  useLocation,
  useNavigate,
} from 'react-router-dom';

import { Link } from 'react-router-dom';
import { Fragment, useCallback, useEffect, useMemo, useState } from 'react';

import { SettingsPage } from './settings';
import { useIsMobile } from './responsive';

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

const TOAST_AUTO_HIDE_MS = 4500;

function ToastList({ toasts }) {
  if (!toasts.length) return null;
  return (
    <div
      role="region"
      aria-label="Notifications"
      style={{
        position: 'fixed',
        bottom: 16,
        right: 16,
        zIndex: 9999,
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        maxWidth: 'min(360px, calc(100vw - 32px))',
        overflowWrap: 'anywhere',
      }}
    >
      {toasts.map((t) => (
        <div
          key={t.id}
          role="alert"
          style={{
            padding: '12px 16px',
            borderRadius: 6,
            background: t.color === 'danger' ? '#bd271e' : t.color === 'warning' ? '#d97706' : '#017d73',
            color: '#fff',
            boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
            fontSize: 14,
          }}
        >
          {t.message}
        </div>
      ))}
    </div>
  );
}

function HeaderTitle({ compact }) {
  const logoSize = compact ? 32 : 64;
  return (
    <Link to="/" style={{ color: 'inherit', textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: 8 }}>
      <img src="/logo-128.png" alt="" width={logoSize} height={logoSize} style={{ display: 'block' }} />
      <span style={compact ? { fontSize: 20, lineHeight: 1.2 } : undefined}>IPTV-Proxy configuration</span>
    </Link>
  );
}

const router = createBrowserRouter([
  {
    path: '/',
    id: 'root',
    element: <Root />,
    children: [
      { index: true, element: <MainPage /> },
      { path: '/settings', element: <SettingsPage /> },
    ],
  },
]);

export default function App() {
  return <RouterProvider router={router} />;
}

function Root() {
  return (
    <PageLayout>
      <Outlet />
    </PageLayout>
  );
}

function PageLayout({ children }) {
  const navigate = useNavigate();
  const location = useLocation();
  const isMobile = useIsMobile();
  const isSettings = location.pathname === '/settings';

  const rightSideItems = (
    <EuiFlexGroup key="header-right" alignItems="center" gutterSize="m" responsive={false}>
      <EuiFlexItem grow={false}>
        {isMobile ? (
          <EuiButtonIcon iconType="gear" onClick={() => navigate('/settings')} display="base" size="m" aria-label="Settings" title="Settings" />
        ) : (
          <EuiButton iconType="gear" onClick={() => navigate('/settings')} size="s">
            Settings
          </EuiButton>
        )}
      </EuiFlexItem>
    </EuiFlexGroup>
  );

  return (
    <EuiPageTemplate panelled={!isMobile}>
      <EuiPageTemplate.Header
        pageTitle={<HeaderTitle compact={isMobile} />}
        rightSideItems={[rightSideItems]}
        responsive={!isMobile}
        alignItems="center"
        paddingSize={isMobile ? 's' : 'l'}
      />
      <EuiPageTemplate.Section restrictWidth={1400} paddingSize={isMobile ? 's' : 'l'}>
        {isSettings && (
          <EuiButtonEmpty iconType="arrowLeft" onClick={() => navigate('/')} flush="left" style={{ paddingLeft: 0 }}>
            Back
          </EuiButtonEmpty>
        )}
        {children}
      </EuiPageTemplate.Section>
    </EuiPageTemplate>
  );
}

const PATTERN_SECTIONS = ['group_inclusions', 'group_exclusions', 'channel_inclusions', 'channel_exclusions'];

function escapeRegexLiteral(s) {
  return (s || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function MainPage() {
  const [selectedTabId, setSelectedTabId] = useState('groups');
  const [channelGroupFilter, setChannelGroupFilter] = useState('');
  const [processingPrepopulate, setProcessingPrepopulate] = useState(null);
  const [showIncluded, setShowIncluded] = useState(true);
  const [showExcluded, setShowExcluded] = useState(true);
  const [showLiveTV, setShowLiveTV] = useState(true);
  const [showVOD, setShowVOD] = useState(true);
  const [processingAddInProgress, setProcessingAddInProgress] = useState(false);
  const [groupsRefreshKey, setGroupsRefreshKey] = useState(0);
  const [channelsRefreshKey, setChannelsRefreshKey] = useState(0);
  const [toasts, setToasts] = useState([]);
  const isMobile = useIsMobile();

  const addToast = useCallback((message, color = 'success') => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, message, color }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, TOAST_AUTO_HIDE_MS);
  }, []);

  const onSettingsSaved = useCallback(() => {
    setGroupsRefreshKey((k) => k + 1);
    setChannelsRefreshKey((k) => k + 1);
  }, []);

  const onSaveGroupReplacement = useCallback((fromName, toName) => {
    if (!fromName || !toName) return Promise.resolve();
    return fetch('/api/settings')
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error(res.statusText))))
      .then((data) => {
        const settings = data.effective ?? data;
        const repl = settings?.replacements ?? {};
        const list = Array.isArray(repl['groups-replacements']) ? [...repl['groups-replacements']] : [];
        const escaped = escapeRegexLiteral(fromName);
        const filtered = list.filter((r) => r.replace !== escaped);
        filtered.push({ replace: escaped, with: toName });
        const next = { ...settings, replacements: { ...repl, 'groups-replacements': filtered } };
        return fetch('/api/settings', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(next) });
      })
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        setGroupsRefreshKey((k) => k + 1);
        setChannelsRefreshKey((k) => k + 1);
        addToast('Group replacement saved');
      });
  }, [addToast]);

  const onSaveChannelReplacement = useCallback((fromName, toName) => {
    if (!fromName || !toName) return Promise.resolve();
    return fetch('/api/settings')
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error(res.statusText))))
      .then((data) => {
        const settings = data.effective ?? data;
        const repl = settings?.replacements ?? {};
        const list = Array.isArray(repl['names-replacements']) ? [...repl['names-replacements']] : [];
        const escaped = escapeRegexLiteral(fromName);
        const filtered = list.filter((r) => r.replace !== escaped);
        filtered.push({ replace: escaped, with: toName });
        const next = { ...settings, replacements: { ...repl, 'names-replacements': filtered } };
        return fetch('/api/settings', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(next) });
      })
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        setGroupsRefreshKey((k) => k + 1);
        setChannelsRefreshKey((k) => k + 1);
        addToast('Channel replacement saved');
      });
  }, [addToast]);

  const tabs = [
    { id: 'groups', name: 'Groups' },
    { id: 'channels', name: 'Channels' },
    { id: 'processing', name: 'Processing' },
    { id: 'watch', name: 'Watch' },
    { id: 'users', name: 'Users' },
  ];

  const onGroupViewChannels = (groupName) => {
    setChannelGroupFilter(groupName || '');
    setSelectedTabId('channels');
  };

  const onAddToProcessing = (opts) => {
    if (PATTERN_SECTIONS.includes(opts.section)) {
      setProcessingAddInProgress(true);
      fetch('/api/settings')
        .then((res) => (res.ok ? res.json() : Promise.reject(new Error(res.statusText))))
        .then((data) => {
          const settings = data.effective ?? data;
          const list = Array.isArray(settings[opts.section]) ? settings[opts.section] : [];
          const escaped = escapeRegexLiteral(opts.value || '').trim();
          if (!escaped || list.includes(escaped)) {
            setGroupsRefreshKey((k) => k + 1);
            setChannelsRefreshKey((k) => k + 1);
            return;
          }
          const next = { ...settings, [opts.section]: [...list, escaped] };
          return fetch('/api/settings', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(next),
          }).then((res) => {
            if (!res.ok) throw new Error(res.statusText);
            setGroupsRefreshKey((k) => k + 1);
            setChannelsRefreshKey((k) => k + 1);
            const label = opts.section.includes('inclusion') ? 'Added to inclusions' : 'Added to exclusions';
            addToast(label);
          });
        })
        .catch(() => {})
        .finally(() => setProcessingAddInProgress(false));
    } else {
      setProcessingPrepopulate(opts);
      setSelectedTabId('processing');
    }
  };

  return (
    <EuiPanel paddingSize={isMobile ? 'm' : 'l'}>
      <EuiTitle size={isMobile ? 's' : 'm'}>
        <h2>Data &amp; processing</h2>
      </EuiTitle>
      <EuiSpacer size={isMobile ? 's' : 'm'} />
      <EuiTabs size={isMobile ? 's' : 'm'}>
        {tabs.map((tab) => (
          <EuiTab key={tab.id} onClick={() => setSelectedTabId(tab.id)} isSelected={selectedTabId === tab.id}>
            {tab.name}
          </EuiTab>
        ))}
      </EuiTabs>
      <EuiSpacer size="m" />
      {selectedTabId === 'groups' && (
        <>
          <EuiFilterGroup style={{ gap: 6, borderRadius: 6 }}>
            <EuiFilterButton
              hasActiveFilters={showIncluded}
              onClick={() => setShowIncluded(!showIncluded)}
              isSelected={showIncluded}
              isToggle
              withNext
              style={{ borderRadius: 6 }}
            >
              Included
            </EuiFilterButton>
            <EuiFilterButton
              hasActiveFilters={showExcluded}
              onClick={() => setShowExcluded(!showExcluded)}
              isSelected={showExcluded}
              isToggle
              style={{ borderRadius: 6 }}
            >
              Excluded
            </EuiFilterButton>
          </EuiFilterGroup>
          <EuiSpacer size="s" />
          <GroupsTab
            showIncluded={showIncluded}
            showExcluded={showExcluded}
            onViewChannels={onGroupViewChannels}
            onAddToProcessing={onAddToProcessing}
            onSaveGroupReplacement={onSaveGroupReplacement}
            addInProgress={processingAddInProgress}
            refreshKey={groupsRefreshKey}
            addToast={addToast}
          />
        </>
      )}
      {selectedTabId === 'channels' && (
        <ChannelsTab
          groupFilter={channelGroupFilter}
          showIncluded={showIncluded}
          showExcluded={showExcluded}
          onShowIncludedChange={setShowIncluded}
          onShowExcludedChange={setShowExcluded}
          onClearGroupFilter={() => setChannelGroupFilter('')}
          onAddToProcessing={onAddToProcessing}
          onSaveChannelReplacement={onSaveChannelReplacement}
          addInProgress={processingAddInProgress}
          refreshKey={channelsRefreshKey}
          addToast={addToast}
        />
      )}
      {selectedTabId === 'processing' && (
        <ProcessingTab
          prepopulate={processingPrepopulate}
          onClearPrepopulate={() => setProcessingPrepopulate(null)}
          addToast={addToast}
          onSettingsSaved={onSettingsSaved}
        />
      )}
      {selectedTabId === 'watch' && <WatchTab addToast={addToast} />}
      {selectedTabId === 'users' && <UsersTab addToast={addToast} />}
      <ToastList toasts={toasts} />
    </EuiPanel>
  );
}

function copyToClipboard(text, addToast) {
  const str = String(text);
  const fallback = () => {
    const ta = document.createElement('textarea');
    ta.value = str;
    ta.setAttribute('readonly', '');
    ta.style.position = 'absolute';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand('copy');
      addToast?.('Copied to clipboard');
    } catch (e) {
      addToast?.('Copy failed', 'danger');
    }
    document.body.removeChild(ta);
  };
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(str).then(() => addToast?.('Copied to clipboard'), fallback);
  } else {
    fallback();
  }
}

function CopyableField({ label, value, addToast }) {
  const copy = () => {
    if (value == null || value === '') return;
    copyToClipboard(value, addToast);
  };
  const displayValue = value != null && value !== '' ? value : '—';
  const canCopy = value != null && value !== '';
  return (
    <EuiFormRow label={label} fullWidth>
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          gap: 8,
          padding: '8px 12px',
          background: 'var(--euiColorEmptyShade)',
          border: '1px solid var(--euiColorLightShade)',
          borderRadius: 6,
        }}
      >
        <EuiToolTip content="Copy to clipboard">
          <EuiButtonIcon
            iconType="copyClipboard"
            aria-label="Copy"
            onClick={copy}
            isDisabled={!canCopy}
            style={{ flexShrink: 0, marginTop: 2 }}
          />
        </EuiToolTip>
        <span
          style={{
            fontFamily: 'monospace',
            fontSize: 13,
            wordBreak: 'break-all',
            flex: 1,
            minWidth: 0,
          }}
        >
          {displayValue}
        </span>
      </div>
    </EuiFormRow>
  );
}

// --- Mobile (collapsed table) helpers ---

// Collapsed tables render only `summary` (one full-width compact card cell); every desktop column is
// hidden there, and `summary` is not rendered on desktop, so the desktop layout stays unchanged.
function withMobileSummary(summary, columns) {
  return [
    { ...summary, mobileOptions: { ...summary.mobileOptions, only: true, header: false, width: '100%' } },
    ...columns.map((c) => ({ ...c, mobileOptions: { ...c.mobileOptions, show: false } })),
  ];
}

// 40px icon button: comfortable touch target for card actions.
function TouchIconButton({ label, ...rest }) {
  return <EuiButtonIcon size="m" aria-label={label} title={label} {...rest} />;
}

// Native link (see AGENTS.md: EuiButtonEmpty with href gets blocked) sized like TouchIconButton.
const TOUCH_LINK_STYLE = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 40,
  height: 40,
  color: 'var(--euiColorPrimary)',
  borderRadius: 4,
};

function TouchActions({ children }) {
  return <span style={{ display: 'inline-flex', alignItems: 'center', flexWrap: 'wrap', justifyContent: 'flex-end' }}>{children}</span>;
}

// Condensed mobile card row: status stripe, optional 32px logo, one-line title and subtitle, trailing actions.
// Status is conveyed by the stripe and row tint, plus screen-reader text; long text is ellipsized (full text in title).
function CompactRow({ excluded, showLogo, logo, title, titleHint, subtitle, actions }) {
  const status = excluded ? 'Excluded' : 'Included';
  const oneLine = { whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' };
  return (
    <div
      data-testid="compact-row"
      title={status}
      style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%', boxSizing: 'border-box', minHeight: 44, paddingLeft: 8, borderLeft: `4px solid ${excluded ? '#bd271e' : '#017d73'}` }}
    >
      {showLogo && (
        <div style={{ flex: '0 0 32px', width: 32, height: 32, borderRadius: 4, background: 'rgba(0,0,0,0.04)', display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden' }}>
          {logo && <img src={logo} alt="" style={{ maxWidth: 32, maxHeight: 32, objectFit: 'contain' }} onError={(e) => { e.target.style.display = 'none'; }} />}
        </div>
      )}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div title={titleHint || title} style={{ ...oneLine, fontWeight: 600, fontSize: 14, lineHeight: '18px' }}>{title}</div>
        <div className="euiTextColor--subdued" title={subtitle} style={{ ...oneLine, fontSize: 12, lineHeight: '16px' }}>
          <EuiScreenReaderOnly><span>{status}. </span></EuiScreenReaderOnly>
          {subtitle}
        </div>
      </div>
      <div style={{ flex: '0 0 auto', display: 'inline-flex', alignItems: 'center' }}>{actions}</div>
    </div>
  );
}

// "More actions" (⋯) popover for secondary card actions; items are 40px tall for touch. Falsy items are skipped.
function RowOverflowMenu({ items }) {
  const [open, setOpen] = useState(false);
  return (
    <EuiPopover
      button={<TouchIconButton iconType="boxesHorizontal" label="More actions" onClick={() => setOpen((o) => !o)} />}
      isOpen={open}
      closePopover={() => setOpen(false)}
      panelPaddingSize="none"
      anchorPosition="downRight"
    >
      <EuiContextMenuPanel
        items={items.filter(Boolean).map((it) => (
          <EuiContextMenuItem
            key={it.label}
            icon={<EuiIcon type={it.icon} color={it.color} />}
            disabled={it.disabled}
            onClick={() => { setOpen(false); it.onClick(); }}
            style={{ minHeight: 40 }}
          >
            {it.label}
          </EuiContextMenuItem>
        ))}
      />
    </EuiPopover>
  );
}

// Collapsed tables lose EUI's sort popover (all sortable columns are hidden), so phones get this instead.
function MobileSortControl({ options, field, direction, onChange }) {
  return (
    <EuiFlexGroup gutterSize="s" alignItems="center" responsive={false}>
      <EuiFlexItem style={{ minWidth: 0 }}>
        <EuiSelect
          fullWidth
          prepend="Sort by"
          aria-label="Sort by"
          options={options}
          value={field}
          onChange={(e) => onChange(e.target.value, direction)}
        />
      </EuiFlexItem>
      <EuiFlexItem grow={false}>
        <TouchIconButton
          display="base"
          iconType={direction === 'asc' ? 'sortUp' : 'sortDown'}
          label={direction === 'asc' ? 'Ascending (tap for descending)' : 'Descending (tap for ascending)'}
          onClick={() => onChange(field, direction === 'asc' ? 'desc' : 'asc')}
        />
      </EuiFlexItem>
    </EuiFlexGroup>
  );
}

// In-place editor with Save/Cancel for names, groups and replacement rules. `touch` uses 40px buttons.
function InlineEditField({ value, onChange, onSave, onCancel, onEscape, cancelColor, touch }) {
  const button = (iconType, label, onClick, color) =>
    touch ? (
      <TouchIconButton iconType={iconType} label={label} onClick={onClick} color={color} />
    ) : (
      <EuiToolTip content={label}>
        <EuiButtonEmpty size="xs" iconType={iconType} color={color} onClick={onClick} aria-label={label} />
      </EuiToolTip>
    );
  return (
    <EuiFlexGroup gutterSize="xs" alignItems="center" responsive={false}>
      <EuiFlexItem grow={true} style={touch ? { minWidth: 0 } : undefined}>
        <EuiFieldText
          fullWidth
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') onSave();
            if (e.key === 'Escape') (onEscape ?? onCancel)();
          }}
          autoFocus
        />
      </EuiFlexItem>
      <EuiFlexItem grow={false}>{button('check', 'Save', onSave, 'primary')}</EuiFlexItem>
      <EuiFlexItem grow={false}>{button('cross', 'Cancel', onCancel, cancelColor)}</EuiFlexItem>
    </EuiFlexGroup>
  );
}

function WatchTab({ addToast }) {
  const [settings, setSettings] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [users, setUsers] = useState([]);
  const [selectedUser, setSelectedUser] = useState('');
  const [watchCreds, setWatchCreds] = useState(null); // { username, password }

  useEffect(() => {
    setLoading(true);
    setError(null);
    Promise.all([
      fetch('/api/settings', { cache: 'no-store' }).then((r) => (r.ok ? r.json() : Promise.reject(new Error(r.statusText)))),
      fetch('/api/users').then((r) => (r.ok ? r.json() : { users: [] })),
    ])
      .then(([settingsData, usersData]) => {
        const s = settingsData.effective ?? settingsData;
        setSettings(s);
        const userList = usersData.users || [];
        setUsers(userList);
        if (userList.length > 0) {
          setSelectedUser(userList[0].username);
        }
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  // Fetch password for selected user.
  useEffect(() => {
    setWatchCreds(null);
    if (!selectedUser) return;
    fetch(`/api/users/${encodeURIComponent(selectedUser)}/watch`)
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => setWatchCreds(data || null))
      .catch(() => setWatchCreds(null));
  }, [selectedUser]);

  const activeUser = watchCreds?.username || settings?.user || '';
  const activePassword = watchCreds?.password || settings?.password || '';

  const watch = useMemo(() => {
    if (!settings) return null;
    const host = settings.hostname || 'localhost';
    const port = Number(settings.advertised_port) || Number(settings.port) || 8080;
    const scheme = settings.https ? 'https' : 'http';
    const customEnd = (settings.custom_endpoint || '').trim().replace(/^\/+|\/+$/g, '');
    const baseUrl = customEnd ? `${scheme}://${host}:${port}/${customEnd}` : `${scheme}://${host}:${port}`;
    const user = activeUser;
    const password = activePassword;
    const m3uFileName = settings.m3u_file_name || 'iptv.m3u';
    const isM3U = !!settings.m3u_url;
    const isXtream = !!settings.xtream_base_url;

    const m3uPlaylistUrl = isM3U
      ? `${baseUrl}/${m3uFileName}?username=${encodeURIComponent(user)}&password=${encodeURIComponent(password)}`
      : '';

    const xtreamBase = isXtream ? baseUrl : '';
    const xtreamGetPhp = isXtream ? `${baseUrl}/get.php?username=${encodeURIComponent(user)}&password=${encodeURIComponent(password)}` : '';
    const xtreamPlayerApi = isXtream ? `${baseUrl}/player_api.php` : '';
    const epgUrl = isXtream ? `${baseUrl}/xmltv.php?username=${encodeURIComponent(user)}&password=${encodeURIComponent(password)}` : '';

    return {
      baseUrl,
      isM3U,
      isXtream,
      m3uPlaylistUrl,
      xtreamBase,
      xtreamGetPhp,
      xtreamPlayerApi,
      epgUrl,
      user,
      password,
    };
  }, [settings, activeUser, activePassword]);

  if (loading) {
    return <EuiCallOut title="Loading…" iconType="refresh" />;
  }
  if (error) {
    return (
      <EuiCallOut title="Error" color="danger" iconType="alert">
        <p>{error}</p>
      </EuiCallOut>
    );
  }
  if (!watch) {
    return null;
  }

  const copySection = (title, description, lines) => {
    const block = [title, description, '', ...lines].join('\n');
    copyToClipboard(block, addToast);
  };

  return (
    <Fragment>
      {users.length > 0 && (
        <EuiFormRow label="Show connection details for user:" style={{ maxWidth: 300, marginBottom: 16 }}>
          <EuiSelect
            options={users.map((u) => ({
              value: u.username,
              text: u.username + (!u.enabled ? ' (disabled)' : ''),
            }))}
            value={selectedUser}
            onChange={(e) => setSelectedUser(e.target.value)}
          />
        </EuiFormRow>
      )}
      <p className="euiTextColor--subdued" style={{ marginBottom: 16 }}>
        Use these URLs and credentials in your IPTV player (e.g. VLC, Kodi, TiviMate). Copy each value with the clipboard button, or copy a full section to share in a messaging app.
      </p>

      {watch.isM3U && (
        <EuiPanel paddingSize="m" style={{ marginBottom: 16 }}>
          <EuiFlexGroup justifyContent="spaceBetween" alignItems="center" responsive={false} wrap>
            <EuiFlexItem grow={false}>
              <EuiTitle size="xs">
                <h3 style={{ marginTop: 0 }}>M3U playlist</h3>
              </EuiTitle>
            </EuiFlexItem>
            <EuiFlexItem grow={false}>
              <EuiButtonEmpty
                size="xs"
                iconType="copyClipboard"
                onClick={() =>
                  copySection(
                    '--- M3U playlist (IPTV-Proxy) ---',
                    'Use this URL in your player as the playlist / M3U URL.',
                    [`Playlist URL: ${watch.m3uPlaylistUrl}`]
                  )
                }
              >
                Copy section to share
              </EuiButtonEmpty>
            </EuiFlexItem>
          </EuiFlexGroup>
          <p className="euiTextColor--subdued" style={{ fontSize: 12, marginBottom: 12, marginTop: 4 }}>
            Enter this URL in your player as the playlist / M3U URL.
          </p>
          <CopyableField label="Playlist URL" value={watch.m3uPlaylistUrl} addToast={addToast} />
        </EuiPanel>
      )}

      {watch.isXtream && (
        <EuiPanel paddingSize="m" style={{ marginBottom: 16 }}>
          <EuiFlexGroup justifyContent="spaceBetween" alignItems="center" responsive={false} wrap>
            <EuiFlexItem grow={false}>
              <EuiTitle size="xs">
                <h3 style={{ marginTop: 0 }}>Xtream Codes</h3>
              </EuiTitle>
            </EuiFlexItem>
            <EuiFlexItem grow={false}>
              <EuiButtonEmpty
                size="xs"
                iconType="copyClipboard"
                onClick={() =>
                  copySection(
                    '--- Xtream Codes (IPTV-Proxy) ---',
                    'Use in Xtream-compatible players (TiviMate, IPTV Smarters). Server URL = base; use username and password below.',
                    [
                      `Server URL: ${watch.xtreamBase}`,
                      `get.php: ${watch.xtreamGetPhp}`,
                      `player_api.php: ${watch.xtreamPlayerApi}`,
                      `Username: ${watch.user}`,
                      `Password: ${watch.password}`,
                    ]
                  )
                }
              >
                Copy section to share
              </EuiButtonEmpty>
            </EuiFlexItem>
          </EuiFlexGroup>
          <p className="euiTextColor--subdued" style={{ fontSize: 12, marginBottom: 12, marginTop: 4 }}>
            Use these in Xtream-compatible players (TiviMate, IPTV Smarters, etc.). Server URL = base URL; username and password = proxy credentials.
          </p>
          <CopyableField label="Server URL (base)" value={watch.xtreamBase} addToast={addToast} />
          <EuiSpacer size="s" />
          <CopyableField label="get.php (playlist)" value={watch.xtreamGetPhp} addToast={addToast} />
          <EuiSpacer size="s" />
          <CopyableField label="player_api.php" value={watch.xtreamPlayerApi} addToast={addToast} />
          <EuiSpacer size="s" />
          <CopyableField label="Username" value={watch.user} addToast={addToast} />
          <EuiSpacer size="s" />
          <CopyableField label="Password" value={watch.password} addToast={addToast} />
        </EuiPanel>
      )}

      <EuiPanel paddingSize="m">
        <EuiFlexGroup justifyContent="spaceBetween" alignItems="center" responsive={false} wrap>
          <EuiFlexItem grow={false}>
            <EuiTitle size="xs">
              <h3 style={{ marginTop: 0 }}>EPG / XMLTV</h3>
            </EuiTitle>
          </EuiFlexItem>
          {watch.isXtream && (
            <EuiFlexItem grow={false}>
              <EuiButtonEmpty
                size="xs"
                iconType="copyClipboard"
                onClick={() =>
                  copySection(
                    '--- EPG / XMLTV (IPTV-Proxy) ---',
                    'Use this URL in your player as the EPG / guide source.',
                    [`EPG URL: ${watch.epgUrl}`]
                  )
                }
              >
                Copy section to share
              </EuiButtonEmpty>
            </EuiFlexItem>
          )}
        </EuiFlexGroup>
        {watch.isXtream ? (
          <Fragment>
            <p className="euiTextColor--subdued" style={{ fontSize: 12, marginBottom: 12, marginTop: 4 }}>
              Use this URL in your player as the EPG / guide source.
            </p>
            <CopyableField label="EPG URL" value={watch.epgUrl} addToast={addToast} />
          </Fragment>
        ) : (
          <p className="euiTextColor--subdued" style={{ fontSize: 12, marginTop: 4 }}>
            {watch.isM3U
              ? 'EPG is not provided by the proxy in M3U mode. If your playlist includes tvg-url tags, your player may use those for the guide.'
              : 'Configure an M3U URL or Xtream credentials to see player URLs here.'}
          </p>
        )}
      </EuiPanel>
    </Fragment>
  );
}

function GroupsTab({ showIncluded, showExcluded, onViewChannels, onAddToProcessing, onSaveGroupReplacement, addInProgress, refreshKey, addToast }) {
  const [groups, setGroups] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(25);
  const [sortField, setSortField] = useState('name');
  const [sortDirection, setSortDirection] = useState('asc');
  const [search, setSearch] = useState('');
  const [editingGroupName, setEditingGroupName] = useState(null);
  const [editingGroupValue, setEditingGroupValue] = useState('');
  const isMobile = useIsMobile();

  const fetchGroups = useCallback(() => {
    setLoading(true);
    setError(null);
    fetch('/api/groups', { cache: 'no-store' })
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        return res.json();
      })
      .then((data) => setGroups(Array.isArray(data) ? data : []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => fetchGroups(), [fetchGroups, refreshKey]);

  const filtered = useMemo(() => {
    let list = groups;
    const bothUnchecked = !showIncluded && !showExcluded;
    list = list.filter(
      (g) => bothUnchecked || (showIncluded && g.excluded !== true) || (showExcluded && g.excluded === true)
    );
    if (search.trim()) {
      const s = search.toLowerCase();
      list = list.filter((g) => (g.name || '').toLowerCase().includes(s));
    }
    return list;
  }, [groups, search, showIncluded, showExcluded]);

  const sorted = useMemo(() => {
    const out = [...filtered];
    out.sort((a, b) => {
      const va = a[sortField];
      const vb = b[sortField];
      const cmp = typeof va === 'number' && typeof vb === 'number' ? va - vb : String(va ?? '').localeCompare(String(vb ?? ''));
      return sortDirection === 'asc' ? cmp : -cmp;
    });
    return out;
  }, [filtered, sortField, sortDirection]);

  const paginated = useMemo(() => {
    const start = pageIndex * pageSize;
    return sorted.slice(start, start + pageSize);
  }, [sorted, pageIndex, pageSize]);

  const pagination = {
    pageIndex,
    pageSize,
    totalItemCount: sorted.length,
    pageSizeOptions: PAGE_SIZE_OPTIONS,
    showPerPageOptions: true,
    onChange: (pIndex, pSize) => {
      setPageIndex(pIndex);
      setPageSize(pSize);
    },
  };

  const sorting = {
    sort: { field: sortField, direction: sortDirection },
    enableAllColumns: true,
  };

  const onTableChange = ({ page, sort }) => {
    if (page) {
      setPageIndex(page.index);
      setPageSize(page.size);
    }
    if (sort) {
      setSortField(sort.field);
      setSortDirection(sort.direction);
    }
  };

  const getRow = (val, item) => (item && typeof item === 'object' && 'name' in item ? item : val && typeof val === 'object' && 'name' in val ? val : {});
  const getRowIndex = (val, item, idx) => (typeof idx === 'number' ? idx : typeof item === 'number' ? item : 0);

  const saveGroupEdit = () => onSaveGroupReplacement?.(editingGroupName, editingGroupValue)?.then(() => setEditingGroupName(null))?.catch(() => {});
  const cancelGroupEdit = () => { setEditingGroupName(null); addToast?.('Canceled'); };
  const startGroupEdit = (name) => { setEditingGroupName(name); setEditingGroupValue(name); };

  // Mobile card: one condensed row (status stripe, name, channel count); view channels visible, the rest in "More actions".
  const mobileColumn = {
    field: 'name',
    name: 'Group title',
    sortable: false,
    mobileOptions: {
      render: (row) => {
        const displayName = row.name ?? '—';
        const groupName = row.name ?? '';
        const isEditing = editingGroupName === displayName;
        const count = typeof row.channel_count === 'number' ? row.channel_count : 0;
        return (
          <div style={{ width: '100%' }}>
            <CompactRow
              excluded={row.excluded === true}
              title={displayName}
              subtitle={`${count} channels${row.replaced ? ' · replaced' : ''}`}
              actions={
                <Fragment>
                  <TouchIconButton iconType="eye" label="View channels" onClick={() => onViewChannels(groupName)} />
                  <RowOverflowMenu
                    items={[
                      !isEditing && { label: 'Edit', icon: 'pencil', onClick: () => startGroupEdit(displayName) },
                      { label: 'Add to inclusions', icon: 'plusInCircleFilled', color: 'success', disabled: addInProgress, onClick: () => onAddToProcessing({ section: 'group_inclusions', value: groupName }) },
                      { label: 'Add to exclusions', icon: 'minusInCircleFilled', color: 'danger', disabled: addInProgress, onClick: () => onAddToProcessing({ section: 'group_exclusions', value: groupName }) },
                    ]}
                  />
                </Fragment>
              }
            />
            {isEditing && (
              <InlineEditField value={editingGroupValue} onChange={setEditingGroupValue} onSave={saveGroupEdit} onCancel={cancelGroupEdit} onEscape={() => setEditingGroupName(null)} cancelColor="danger" touch />
            )}
          </div>
        );
      },
    },
  };

  const columns = [
    {
      name: '#',
      width: '60px',
      render: (val, item, idx) => {
        const i = getRowIndex(val, item, idx);
        const p = Number(pageIndex);
        const ps = Number(pageSize);
        const n = (Number.isNaN(p) ? 0 : p) * (Number.isNaN(ps) ? 25 : ps) + i + 1;
        return Number.isNaN(n) ? i + 1 : n;
      },
    },
    {
      field: 'name',
      name: 'Group title',
      sortable: true,
      truncateText: true,
      render: (name, item) => {
        const row = getRow(name, item);
        const displayName = row.name ?? name ?? '—';
        const isEditing = editingGroupName === displayName;
        return (
          <EuiFlexGroup gutterSize="xs" alignItems="center" wrap>
            <EuiFlexItem grow={true}>
              {isEditing ? (
                <InlineEditField value={editingGroupValue} onChange={setEditingGroupValue} onSave={saveGroupEdit} onCancel={cancelGroupEdit} onEscape={() => setEditingGroupName(null)} cancelColor="danger" />
              ) : (
                <span>{displayName}</span>
              )}
            </EuiFlexItem>
            {!isEditing && (
              <EuiFlexItem grow={false}>
                <EuiToolTip content="Edit (add replacement)">
                  <EuiButtonEmpty size="xs" iconType="pencil" onClick={() => { setEditingGroupName(displayName); setEditingGroupValue(displayName); }} aria-label="Edit" />
                </EuiToolTip>
              </EuiFlexItem>
            )}
            {row.replaced && (
              <EuiFlexItem grow={false}>
                <EuiBadge color="hollow" title="Value was replaced by a rule">Replaced</EuiBadge>
              </EuiFlexItem>
            )}
          </EuiFlexGroup>
        );
      },
    },
    {
      field: 'excluded',
      name: 'Status',
      width: '100px',
      sortable: true,
      render: (excluded, item) => {
        const row = getRow(excluded, item);
        const ex = row.excluded === true;
        return (
          <EuiFlexGroup gutterSize="xs" alignItems="center">
            <EuiFlexItem grow={false}>
              <EuiIcon type={ex ? 'crossInCircle' : 'checkInCircleFilled'} color={ex ? 'danger' : 'success'} title={ex ? 'Will be excluded from output' : 'Will be included in output'} />
            </EuiFlexItem>
            <EuiFlexItem grow={false}>{ex ? 'Excluded' : 'Included'}</EuiFlexItem>
          </EuiFlexGroup>
        );
      },
    },
    {
      field: 'channel_count',
      name: 'Channels',
      sortable: true,
      width: '100px',
      dataType: 'number',
      render: (val) => (typeof val === 'number' ? val : 0),
    },
    {
      name: 'Actions',
      width: '160px',
      render: (val, item) => {
        const row = getRow(val, item);
        const groupName = row.name ?? '';
        return (
          <div style={{ display: 'inline-flex', alignItems: 'center', gap: 0 }}>
            <EuiToolTip content="View channels">
              <EuiButtonEmpty iconType="eye" size="xs" onClick={() => onViewChannels(groupName)} aria-label="View channels" />
            </EuiToolTip>
            <EuiToolTip content="Add to inclusions">
              <EuiButtonEmpty iconType="plusInCircleFilled" size="xs" color="success" onClick={() => onAddToProcessing({ section: 'group_inclusions', value: groupName })} aria-label="Add to inclusions" isDisabled={addInProgress} />
            </EuiToolTip>
            <EuiToolTip content="Add to exclusions">
              <EuiButtonEmpty iconType="minusInCircleFilled" size="xs" color="danger" onClick={() => onAddToProcessing({ section: 'group_exclusions', value: groupName })} aria-label="Add to exclusions" isDisabled={addInProgress} />
            </EuiToolTip>
          </div>
        );
      },
    },
  ];

  if (error) {
    return (
      <EuiCallOut title="Error" color="danger" iconType="alert">
        <p>{error}</p>
      </EuiCallOut>
    );
  }

  return (
    <Fragment>
      <p className="euiTextColor--subdued">Unique group-title values from the playlist. Use quick links to add to Processing (inclusions, exclusions, replacements).</p>
      <EuiSpacer size="m" />
      <EuiFieldSearch
        placeholder="Filter groups…"
        value={search}
        onChange={(e) => {
          setSearch(e.target.value);
          setPageIndex(0);
        }}
        fullWidth
        isClearable
      />
      <EuiSpacer size="m" />
      {isMobile && (
        <MobileSortControl
          options={[
            { value: 'name', text: 'Group title' },
            { value: 'excluded', text: 'Status' },
            { value: 'channel_count', text: 'Channels' },
          ]}
          field={sortField}
          direction={sortDirection}
          onChange={(f, d) => { setSortField(f); setSortDirection(d); }}
        />
      )}
      <EuiBasicTable
        items={paginated}
        columns={withMobileSummary(mobileColumn, columns)}
        loading={loading}
        noItemsMessage="No groups (no M3U loaded or playlist empty)."
        pagination={pagination}
        sorting={sorting}
        onChange={onTableChange}
        rowProps={(item) => ({ className: item.excluded === true ? 'euiTableRow--excluded' : 'euiTableRow--included' })}
      />
    </Fragment>
  );
}

function ChannelsTab({ groupFilter, showIncluded, showExcluded, onShowIncludedChange, onShowExcludedChange, onClearGroupFilter, onAddToProcessing, onSaveChannelReplacement, addInProgress, refreshKey, addToast }) {
  const [channels, setChannels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(25);
  const [sortField, setSortField] = useState('name');
  const [sortDirection, setSortDirection] = useState('asc');
  const [search, setSearch] = useState('');
  // Type filter: one checkbox per type found in channel URLs (e.g. live, series, movies). Default all on.
  const [typeFilters, setTypeFilters] = useState({});
  const [editingChannelName, setEditingChannelName] = useState(null);
  const [editingChannelValue, setEditingChannelValue] = useState('');
  const isMobile = useIsMobile();

  const fetchChannels = useCallback(() => {
    setLoading(true);
    setError(null);
    fetch('/api/channels', { cache: 'no-store' })
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        return res.json();
      })
      .then((data) => setChannels(Array.isArray(data) ? data : []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => fetchChannels(), [fetchChannels, refreshKey]);
  useEffect(() => {
    if (groupFilter) setPageIndex(0);
  }, [groupFilter]);

  const uniqueTypes = useMemo(() => {
    const set = new Set(channels.map((c) => ((c.type || 'live') + '').toLowerCase()).filter(Boolean));
    return [...set].sort();
  }, [channels]);

  useEffect(() => {
    setTypeFilters((prev) => {
      const next = { ...prev };
      uniqueTypes.forEach((t) => {
        if (next[t] === undefined) next[t] = true;
      });
      return next;
    });
  }, [uniqueTypes.join(',')]);

  const filtered = useMemo(() => {
    let list = channels;
    const bothUnchecked = !showIncluded && !showExcluded;
    list = list.filter(
      (c) => bothUnchecked || (showIncluded && c.excluded !== true) || (showExcluded && c.excluded === true)
    );
    list = list.filter((c) => {
      const t = ((c.type || 'live') + '').toLowerCase();
      return typeFilters[t] !== false;
    });
    if (groupFilter) {
      list = list.filter((c) => (c.group || '') === (groupFilter || ''));
    }
    if (search.trim()) {
      const s = search.toLowerCase();
      list = list.filter(
        (c) =>
          (c.name || '').toLowerCase().includes(s) ||
          (c.tvg_name || '').toLowerCase().includes(s) ||
          (c.group || '').toLowerCase().includes(s)
      );
    }
    return list;
  }, [channels, groupFilter, search, showIncluded, showExcluded, typeFilters]);

  const sorted = useMemo(() => {
    const out = [...filtered];
    out.sort((a, b) => {
      const va = a[sortField];
      const vb = b[sortField];
      const cmp = typeof va === 'number' && typeof vb === 'number' ? va - vb : String(va ?? '').localeCompare(String(vb ?? ''));
      return sortDirection === 'asc' ? cmp : -cmp;
    });
    return out;
  }, [filtered, sortField, sortDirection]);

  const paginated = useMemo(() => {
    const start = pageIndex * pageSize;
    return sorted.slice(start, start + pageSize);
  }, [sorted, pageIndex, pageSize]);

  const pagination = {
    pageIndex,
    pageSize,
    totalItemCount: sorted.length,
    pageSizeOptions: PAGE_SIZE_OPTIONS,
    showPerPageOptions: true,
    onChange: (pIndex, pSize) => {
      setPageIndex(pIndex);
      setPageSize(pSize);
    },
  };

  const sorting = {
    sort: { field: sortField, direction: sortDirection },
    enableAllColumns: true,
  };

  const onChannelsTableChange = ({ page, sort }) => {
    if (page) {
      setPageIndex(page.index);
      setPageSize(page.size);
    }
    if (sort) {
      setSortField(sort.field);
      setSortDirection(sort.direction);
    }
  };

  const getChannelRow = (val, item) => (item && typeof item === 'object' && 'name' in item ? item : val && typeof val === 'object' && 'name' in val ? val : {});

  const saveChannelEdit = () => onSaveChannelReplacement?.(editingChannelName, editingChannelValue)?.then(() => setEditingChannelName(null))?.catch(() => {});
  const cancelChannelEdit = () => { setEditingChannelName(null); addToast?.('Canceled'); };
  const startChannelEdit = (name) => { setEditingChannelName(name); setEditingChannelValue(name); };

  // Mobile card: one condensed row (status stripe, 32px logo, name, group · type); play visible, the rest in "More actions".
  // tvg-name (when different) is only in the name tooltip; empty tvg-id and labels are omitted.
  const mobileColumn = {
    field: 'name',
    name: 'Name',
    sortable: false,
    mobileOptions: {
      render: (r) => {
        const displayName = r.name ?? '—';
        const channelName = r.name ?? '';
        const isEditing = editingChannelName === displayName;
        const logo = typeof r.tvg_logo === 'string' && /^https?:\/\//.test(r.tvg_logo) ? r.tvg_logo : '';
        const subtitle = [r.group || '—', r.type || 'live', (r.name_replaced || r.group_replaced) && 'replaced'].filter(Boolean).join(' · ');
        return (
          <div style={{ width: '100%' }}>
            <CompactRow
              excluded={r.excluded === true}
              showLogo
              logo={logo}
              title={displayName}
              titleHint={r.tvg_name && r.tvg_name !== displayName ? `${displayName}\ntvg-name: ${r.tvg_name}` : displayName}
              subtitle={subtitle}
              actions={
                <Fragment>
                  {r.stream_url && (
                    <a href={r.stream_url} target="_blank" rel="noopener noreferrer" aria-label="Open stream" title={r.stream_url} style={TOUCH_LINK_STYLE}>
                      <EuiIcon type="play" size="m" />
                    </a>
                  )}
                  <RowOverflowMenu
                    items={[
                      !isEditing && { label: 'Edit', icon: 'pencil', onClick: () => startChannelEdit(displayName) },
                      { label: 'Add to inclusions', icon: 'plusInCircleFilled', color: 'success', disabled: addInProgress, onClick: () => onAddToProcessing({ section: 'channel_inclusions', value: channelName }) },
                      { label: 'Add to exclusions', icon: 'minusInCircleFilled', color: 'danger', disabled: addInProgress, onClick: () => onAddToProcessing({ section: 'channel_exclusions', value: channelName }) },
                    ]}
                  />
                </Fragment>
              }
            />
            {isEditing && (
              <InlineEditField value={editingChannelValue} onChange={setEditingChannelValue} onSave={saveChannelEdit} onCancel={cancelChannelEdit} onEscape={() => setEditingChannelName(null)} cancelColor="danger" touch />
            )}
          </div>
        );
      },
    },
  };

  const columns = [
    {
      field: 'tvg_logo',
      name: 'Logo',
      width: '80px',
      render: (url) => {
        if (!url) return '—';
        const isHttp = typeof url === 'string' && (url.startsWith('http://') || url.startsWith('https://'));
        if (!isHttp) return <span title={url}>—</span>;
        return (
          <EuiToolTip content={url} position="top">
            <a href={url} target="_blank" rel="noopener noreferrer" style={{ display: 'inline-block' }}>
              <img
                src={url}
                alt=""
                style={{ maxWidth: 48, maxHeight: 48, objectFit: 'contain', verticalAlign: 'middle' }}
                onError={(e) => {
                  e.target.style.display = 'none';
                }}
              />
            </a>
          </EuiToolTip>
        );
      },
    },
    {
      field: 'name',
      name: 'Name',
      sortable: true,
      truncateText: true,
      render: (name, row) => {
        const r = getChannelRow(name, row);
        const displayName = r.name ?? name ?? '—';
        const isEditing = editingChannelName === displayName;
        return (
          <EuiFlexGroup gutterSize="xs" alignItems="center" wrap>
            <EuiFlexItem grow={true}>
              {isEditing ? (
                <InlineEditField value={editingChannelValue} onChange={setEditingChannelValue} onSave={saveChannelEdit} onCancel={cancelChannelEdit} onEscape={() => setEditingChannelName(null)} cancelColor="danger" />
              ) : (
                <Fragment>
                  {displayName}
                  {r.tvg_name && r.tvg_name !== displayName && (
                    <Fragment>
                      <br />
                      <span className="euiTextColor--subdued" style={{ fontSize: '12px' }}>tvg-name: {r.tvg_name}</span>
                    </Fragment>
                  )}
                </Fragment>
              )}
            </EuiFlexItem>
            {!isEditing && (
              <EuiFlexItem grow={false}>
                <EuiToolTip content="Edit (add replacement)">
                  <EuiButtonEmpty size="xs" iconType="pencil" onClick={() => { setEditingChannelName(displayName); setEditingChannelValue(displayName); }} aria-label="Edit" />
                </EuiToolTip>
              </EuiFlexItem>
            )}
            {r.name_replaced && (
              <EuiFlexItem grow={false}>
                <EuiBadge color="hollow" title="Name was replaced">Replaced</EuiBadge>
              </EuiFlexItem>
            )}
          </EuiFlexGroup>
        );
      },
    },
    {
      field: 'excluded',
      name: 'Status',
      width: '100px',
      render: (excluded, row) => {
        const r = getChannelRow(excluded, row);
        const ex = r.excluded === true;
        return (
          <EuiFlexGroup gutterSize="xs" alignItems="center">
            <EuiFlexItem grow={false}>
              <EuiIcon type={ex ? 'crossInCircle' : 'checkInCircleFilled'} color={ex ? 'danger' : 'success'} title={ex ? 'Will be excluded from output' : 'Will be included in output'} />
            </EuiFlexItem>
            <EuiFlexItem grow={false}>{ex ? 'Excluded' : 'Included'}</EuiFlexItem>
          </EuiFlexGroup>
        );
      },
    },
    {
      field: 'group',
      name: 'Group',
      sortable: true,
      truncateText: true,
      render: (group, row) => {
        const r = getChannelRow(group, row);
        return (
          <EuiFlexGroup gutterSize="xs" alignItems="center" wrap>
            <EuiFlexItem grow={false}>{r.group ?? group ?? '—'}</EuiFlexItem>
            {r.group_replaced && (
              <EuiFlexItem grow={false}>
                <EuiBadge color="hollow" title="Group was replaced">Replaced</EuiBadge>
              </EuiFlexItem>
            )}
          </EuiFlexGroup>
        );
      },
    },
    { field: 'type', name: 'Type', sortable: true, width: '80px', render: (t) => t || 'live' },
    { field: 'tvg_id', name: 'tvg-id', sortable: true, truncateText: true },
    {
      name: 'Actions',
      width: '160px',
      render: (val, item) => {
        const row = getChannelRow(val, item);
        const channelName = row.name ?? '';
        const streamUrl = row.stream_url;
        return (
          <div style={{ display: 'inline-flex', alignItems: 'center', gap: 0 }}>
            <EuiToolTip content="Add to inclusions">
              <EuiButtonEmpty iconType="plusInCircleFilled" size="xs" color="success" onClick={() => onAddToProcessing({ section: 'channel_inclusions', value: channelName })} aria-label="Add to inclusions" isDisabled={addInProgress} />
            </EuiToolTip>
            <EuiToolTip content="Add to exclusions">
              <EuiButtonEmpty iconType="minusInCircleFilled" size="xs" color="danger" onClick={() => onAddToProcessing({ section: 'channel_exclusions', value: channelName })} aria-label="Add to exclusions" isDisabled={addInProgress} />
            </EuiToolTip>
            {streamUrl && (
              <EuiToolTip content={streamUrl}>
                <a
                  href={streamUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label="Open stream"
                  data-testid="channel-open-stream"
                  title={streamUrl}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    padding: '4px 6px',
                    marginLeft: 2,
                    color: 'var(--euiColorPrimary)',
                    borderRadius: 4,
                  }}
                >
                  <EuiIcon type="play" size="s" />
                </a>
              </EuiToolTip>
            )}
          </div>
        );
      },
    },
  ];

  if (error) {
    return (
      <EuiCallOut title="Error" color="danger" iconType="alert">
        <p>{error}</p>
      </EuiCallOut>
    );
  }

  const statusFilterGroup = (
    <EuiFilterGroup style={{ gap: 6, borderRadius: 6 }}>
      <EuiFilterButton
        hasActiveFilters={showIncluded}
        onClick={() => onShowIncludedChange?.(!showIncluded)}
        isSelected={showIncluded}
        isToggle
        withNext
        style={{ borderRadius: 6 }}
      >
        Included
      </EuiFilterButton>
      <EuiFilterButton
        hasActiveFilters={showExcluded}
        onClick={() => onShowExcludedChange?.(!showExcluded)}
        isSelected={showExcluded}
        isToggle
        style={{ borderRadius: 6 }}
      >
        Excluded
      </EuiFilterButton>
    </EuiFilterGroup>
  );

  const typeFilterGroup = uniqueTypes.length > 0 ? (
    <EuiFilterGroup style={{ gap: 6, borderRadius: 6 }}>
      {uniqueTypes.map((t) => (
        <EuiFilterButton
          key={t}
          hasActiveFilters={typeFilters[t] !== false}
          onClick={() => setTypeFilters((f) => ({ ...f, [t]: !(f[t] !== false) }))}
          isSelected={typeFilters[t] !== false}
          isToggle
          withNext={t !== uniqueTypes[uniqueTypes.length - 1]}
          style={{ borderRadius: 6 }}
        >
          {t.charAt(0).toUpperCase() + t.slice(1)}
        </EuiFilterButton>
      ))}
    </EuiFilterGroup>
  ) : null;

  return (
    <Fragment>
      <EuiFlexGroup alignItems="center" gutterSize="m" wrap>
        <EuiFlexItem grow={false}>{statusFilterGroup}</EuiFlexItem>
        {typeFilterGroup && <EuiFlexItem grow={false}>{typeFilterGroup}</EuiFlexItem>}
      </EuiFlexGroup>
      <EuiSpacer size="s" />
      {groupFilter && (
        <Fragment>
          <EuiSpacer size="s" />
          <EuiCallOut title={`Filtered by group: ${groupFilter}`} size="s">
            <EuiButtonEmpty size="xs" onClick={onClearGroupFilter}>Show all</EuiButtonEmpty>
          </EuiCallOut>
        </Fragment>
      )}
      <EuiSpacer size="m" />
      <EuiFieldSearch
        placeholder="Filter by name, tvg-name, or group…"
        value={search}
        onChange={(e) => {
          setSearch(e.target.value);
          setPageIndex(0);
        }}
        fullWidth
        isClearable
      />
      <EuiSpacer size="m" />
      {isMobile && (
        <MobileSortControl
          options={[
            { value: 'name', text: 'Name' },
            { value: 'group', text: 'Group' },
            { value: 'excluded', text: 'Status' },
            { value: 'type', text: 'Type' },
            { value: 'tvg_id', text: 'tvg-id' },
          ]}
          field={sortField}
          direction={sortDirection}
          onChange={(f, d) => { setSortField(f); setSortDirection(d); }}
        />
      )}
      <EuiBasicTable
        items={paginated}
        columns={withMobileSummary(mobileColumn, columns)}
        loading={loading}
        noItemsMessage={groupFilter ? `No channels in group "${groupFilter}".` : 'No channels.'}
        pagination={pagination}
        sorting={sorting}
        onChange={onChannelsTableChange}
        rowProps={(item) => ({ className: item.excluded === true ? 'euiTableRow--excluded' : 'euiTableRow--included' })}
      />
    </Fragment>
  );
}

// Processing order diagram (text) and explanation
const PROCESSING_DIAGRAM = `
  Playlist → [ 1. Inclusions ] → [ 2. Exclusions ] → [ 3. Replacements ] → Output
`;

function ProcessingTab({ prepopulate, onClearPrepopulate, addToast, onSettingsSaved }) {
  const [settings, setSettings] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState(null);
  const [error, setError] = useState(null);
  const [processingSubTab, setProcessingSubTab] = useState('replacements'); // 'replacements' | 'inclusions'
  const [inclusionsSection, setInclusionsSection] = useState('group_inclusions'); // group_inclusions | group_exclusions | channel_inclusions | channel_exclusions
  const [replacementsSection, setReplacementsSection] = useState('global');
  const [newReplace, setNewReplace] = useState('');
  const [newWith, setNewWith] = useState('');
  const [newPattern, setNewPattern] = useState('');
  const [activePatternSection, setActivePatternSection] = useState('group_inclusions');
  const [replacementEdit, setReplacementEdit] = useState(null);
  const [replacementEditValue, setReplacementEditValue] = useState('');
  const isMobile = useIsMobile();

  const fetchSettings = useCallback(() => {
    setLoading(true);
    setError(null);
    fetch('/api/settings')
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        return res.json();
      })
      .then((data) => setSettings(data.effective ?? data))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  useEffect(() => {
    if (prepopulate?.value) {
      setNewPattern(escapeRegexLiteral(prepopulate.value));
      const section = prepopulate.section || 'group_inclusions';
      setActivePatternSection(section);
      if (PATTERN_SECTIONS.includes(prepopulate.section)) {
        setProcessingSubTab('inclusions');
        setInclusionsSection(section);
      }
    }
  }, [prepopulate]);

  useEffect(() => {
    setReplacementEdit(null);
    setReplacementEditValue('');
  }, [replacementsSection]);

  const updateSettings = (updater) => {
    setSettings((prev) => {
      const next = prev ? { ...prev } : {};
      updater(next);
      return next;
    });
  };

  const saveAll = () => {
    if (!settings) return;
    setSaving(true);
    setMessage(null);
    const toSave = stripEmptyReplacementRules(settings);
    fetch('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(toSave),
    })
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        onSettingsSaved?.();
        addToast?.('Saved');
        onClearPrepopulate?.();
      })
      .catch((e) => setMessage('Error: ' + e.message))
      .finally(() => setSaving(false));
  };

  const replacements = settings?.replacements || { 'global-replacements': [], 'names-replacements': [], 'groups-replacements': [] };
  const groupInclusions = settings?.group_inclusions || [];
  const groupExclusions = settings?.group_exclusions || [];
  const channelInclusions = settings?.channel_inclusions || [];
  const channelExclusions = settings?.channel_exclusions || [];

  const addReplacement = () => {
    const key = replacementsSection + '-replacements';
    const list = replacements[key] || [];
    const nextList = [...list, { replace: newReplace, with: newWith }];
    const nextRepl = { ...(settings?.replacements || {}), [key]: nextList };
    const next = { ...settings, replacements: nextRepl };
    setSettings(next);
    setNewReplace('');
    setNewWith('');
    persistSettings(next, 'Rule added');
  };

  const removeReplacement = (key, idx) => {
    if (typeof idx !== 'number' || idx < 0) return;
    const list = (settings?.replacements?.[key] || []).filter((_, i) => i !== idx);
    const nextRepl = { ...(settings?.replacements || {}), [key]: list };
    const next = { ...settings, replacements: nextRepl };
    setSettings(next);
    if (replacementEdit?.key === key && replacementEdit?.rowIndex === idx) setReplacementEdit(null);
    persistSettings(next, 'Rule removed');
  };

  const startReplacementEdit = (key, rowIndex, field, currentValue) => {
    setReplacementEdit({ key, rowIndex, field });
    setReplacementEditValue(currentValue ?? '');
  };

  const stripEmptyReplacementRules = (s) => {
    if (!s?.replacements) return s;
    const next = { ...s, replacements: {} };
    for (const k of ['global-replacements', 'names-replacements', 'groups-replacements']) {
      const list = s.replacements[k] || [];
      next.replacements[k] = list.filter((r) => (r.replace ?? '').trim() !== '' || (r.with ?? '').trim() !== '');
    }
    return next;
  };

  const persistSettings = useCallback((payload, successMessage = 'Saved') => {
    let toSave = payload ?? settings;
    if (!toSave) return Promise.resolve();
    toSave = stripEmptyReplacementRules(toSave);
    return fetch('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(toSave),
    })
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        onSettingsSaved?.();
        addToast?.(successMessage);
        onClearPrepopulate?.();
      })
      .catch((e) => setMessage('Error: ' + e.message));
  }, [settings, onClearPrepopulate, onSettingsSaved, addToast]);

  const applyReplacementEdit = () => {
    if (!replacementEdit || !settings?.replacements) return;
    const { key, rowIndex, field } = replacementEdit;
    const list = [...(settings.replacements[key] || [])];
    if (!list[rowIndex]) {
      setReplacementEdit(null);
      setReplacementEditValue('');
      return;
    }
    list[rowIndex] = { ...list[rowIndex], [field]: replacementEditValue };
    const nextSettings = { ...settings, replacements: { ...settings.replacements, [key]: list } };
    setSettings(nextSettings);
    setReplacementEdit(null);
    setReplacementEditValue('');
    persistSettings(nextSettings, 'Saved');
  };

  const cancelReplacementEdit = () => {
    setReplacementEdit(null);
    setReplacementEditValue('');
    addToast?.('Canceled');
  };

  const addPattern = (listKey) => {
    if (!newPattern.trim()) return;
    const list = settings?.[listKey] || [];
    const nextList = [...list, newPattern.trim()];
    const next = { ...settings, [listKey]: nextList };
    setSettings(next);
    setNewPattern('');
    onClearPrepopulate?.();
    persistSettings(next, 'Pattern added');
  };

  const removePattern = (listKey, idx) => {
    if (typeof idx !== 'number' || idx < 0) return;
    const list = (settings?.[listKey] || []).filter((_, i) => i !== idx);
    const next = { ...settings, [listKey]: list };
    setSettings(next);
    persistSettings(next, 'Pattern removed');
  };

  const patternSectionOptions = [
    { value: 'group_inclusions', text: 'Group inclusions' },
    { value: 'group_exclusions', text: 'Group exclusions' },
    { value: 'channel_inclusions', text: 'Channel inclusions' },
    { value: 'channel_exclusions', text: 'Channel exclusions' },
  ];

  const onRemovePatternClick = (e) => {
    const listKey = e?.currentTarget?.getAttribute?.('data-listkey');
    const idx = parseInt(e?.currentTarget?.getAttribute?.('data-idx'), 10);
    if (listKey != null && !Number.isNaN(idx) && idx >= 0) removePattern(listKey, idx);
  };

  const renderPatternTable = (title, listKey, description) => {
    const list = settings?.[listKey] || [];
    const items = list.map((p, i) => ({ pattern: p, _idx: i }));
    return (
      <Fragment key={listKey}>
        {title ? <EuiTitle size="xs"><h4>{title}</h4></EuiTitle> : null}
        {description ? <p className="euiTextColor--subdued" style={{ fontSize: '12px' }}>{description}</p> : null}
        {(title || description) ? <EuiSpacer size="xs" /> : null}
        <EuiBasicTable
          items={items}
          columns={withMobileSummary({
            field: 'pattern',
            name: 'Pattern (regex)',
            mobileOptions: {
              render: (item) => (
                <EuiFlexGroup gutterSize="s" alignItems="center" justifyContent="spaceBetween" responsive={false} style={{ width: '100%' }}>
                  <EuiFlexItem style={{ minWidth: 0 }}>
                    <code style={{ overflowWrap: 'anywhere' }}>{item.pattern}</code>
                  </EuiFlexItem>
                  <EuiFlexItem grow={false}>
                    <TouchIconButton type="button" color="danger" iconType="trash" label="Remove pattern" data-listkey={listKey} data-idx={item._idx} onClick={onRemovePatternClick} />
                  </EuiFlexItem>
                </EuiFlexGroup>
              ),
            },
          }, [
            { field: 'pattern', name: 'Pattern (regex)' },
            {
              name: 'Actions',
              width: '90px',
              render: (cellValue, item) => {
                const row = item ?? cellValue;
                const idx = row != null && typeof row._idx === 'number' ? row._idx : -1;
                if (idx < 0) return <span />;
                return (
                  <span style={{ display: 'inline-flex' }}>
                    <EuiToolTip content="Remove pattern">
                      <EuiButtonIcon
                        type="button"
                        size="s"
                        color="danger"
                        iconType="trash"
                        data-listkey={listKey}
                        data-idx={idx}
                        onClick={onRemovePatternClick}
                        aria-label="Remove pattern"
                      />
                    </EuiToolTip>
                  </span>
                );
              },
            },
          ])}
          noItemsMessage="No patterns."
        />
        <EuiSpacer size="s" />
      </Fragment>
    );
  };

  if (error) {
    return (
      <EuiCallOut title="Error" color="danger" iconType="alert">
        <p>{error}</p>
      </EuiCallOut>
    );
  }

  const replacementsContent = (
    <Fragment>
      <EuiTabs size="s">
        {['global', 'names', 'groups'].map((id) => (
          <EuiTab key={id} onClick={() => setReplacementsSection(id)} isSelected={replacementsSection === id}>
            {id === 'global' ? 'Global' : id === 'names' ? 'Channels' : 'Groups'}
          </EuiTab>
        ))}
      </EuiTabs>
      <EuiSpacer size="s" />
      {(() => {
        const key = replacementsSection + '-replacements';
        const rules = replacements[key] || [];
        const itemsWithIdx = rules.map((r, i) => ({ ...r, _idx: i })).filter((r) => (r.replace ?? '').trim() !== '' || (r.with ?? '').trim() !== '');
        const isEditing = (idx, field) =>
          replacementEdit?.key === key && replacementEdit?.rowIndex === idx && replacementEdit?.field === field;
        // Cell for one rule field (replace/with): value + pencil, or the inline editor. `touch` = mobile card.
        const renderRuleCell = (field, editLabel, touch) => (val, row) => {
          const idx = row != null && typeof row._idx === 'number' ? row._idx : -1;
          const editing = isEditing(idx, field);
          return (
            <EuiFlexGroup gutterSize="xs" alignItems="center" responsive={false} wrap>
              <EuiFlexItem grow={true} style={touch ? { minWidth: 0 } : undefined}>
                {editing ? (
                  <InlineEditField value={replacementEditValue} onChange={setReplacementEditValue} onSave={applyReplacementEdit} onCancel={cancelReplacementEdit} touch={touch} />
                ) : (
                  <span style={touch ? { fontFamily: 'monospace', overflowWrap: 'anywhere' } : undefined}>{val ?? ''}</span>
                )}
              </EuiFlexItem>
              {!editing && (
                <EuiFlexItem grow={false}>
                  {touch ? (
                    <TouchIconButton iconType="pencil" label={editLabel} onClick={() => startReplacementEdit(key, idx, field, val)} />
                  ) : (
                    <EuiToolTip content="Edit">
                      <EuiButtonEmpty size="xs" iconType="pencil" onClick={() => startReplacementEdit(key, idx, field, val)} aria-label={editLabel} />
                    </EuiToolTip>
                  )}
                </EuiFlexItem>
              )}
            </EuiFlexGroup>
          );
        };
        const onRemoveRuleClick = (e) => {
          const k = e?.currentTarget?.getAttribute?.('data-repkey');
          const i = parseInt(e?.currentTarget?.getAttribute?.('data-idx'), 10);
          if (k != null && !Number.isNaN(i) && i >= 0) removeReplacement(k, i);
        };
        const mobileRuleColumn = {
          field: 'replace',
          name: 'Rule',
          mobileOptions: {
            render: (item) => (
              <EuiFlexGroup gutterSize="xs" alignItems="flexStart" responsive={false} style={{ width: '100%' }}>
                <EuiFlexItem style={{ minWidth: 0 }}>
                  <div className="euiTextColor--subdued" style={{ fontSize: 12 }}>Replace (regex)</div>
                  {renderRuleCell('replace', 'Edit replace pattern', true)(item.replace, item)}
                  <div className="euiTextColor--subdued" style={{ fontSize: 12 }}>With</div>
                  {renderRuleCell('with', 'Edit with value', true)(item.with, item)}
                </EuiFlexItem>
                <EuiFlexItem grow={false}>
                  <TouchIconButton type="button" color="danger" iconType="trash" label="Remove rule" data-repkey={key} data-idx={item._idx} onClick={onRemoveRuleClick} />
                </EuiFlexItem>
              </EuiFlexGroup>
            ),
          },
        };
        return (
          <Fragment>
            <EuiBasicTable
              items={itemsWithIdx}
              columns={withMobileSummary(mobileRuleColumn, [
                {
                  field: 'replace',
                  name: 'Replace (regex)',
                  render: renderRuleCell('replace', 'Edit replace pattern', false),
                },
                {
                  field: 'with',
                  name: 'With',
                  render: renderRuleCell('with', 'Edit with value', false),
                },
                {
                  name: 'Actions',
                  width: '90px',
                  render: (cellValue, item) => {
                    const row = item ?? cellValue;
                    const idx = row != null && typeof row._idx === 'number' ? row._idx : -1;
                    if (idx < 0) return <span />;
                    return (
                      <span style={{ display: 'inline-flex' }}>
                        <EuiToolTip content="Remove rule">
                          <EuiButtonIcon
                            type="button"
                            size="s"
                            color="danger"
                            iconType="trash"
                            data-repkey={key}
                            data-idx={idx}
                            onClick={(e) => {
                              const k = e?.currentTarget?.getAttribute?.('data-repkey');
                              const i = parseInt(e?.currentTarget?.getAttribute?.('data-idx'), 10);
                              if (k != null && !Number.isNaN(i) && i >= 0) removeReplacement(k, i);
                            }}
                            aria-label="Remove rule"
                          />
                        </EuiToolTip>
                      </span>
                    );
                  },
                },
              ])}
              noItemsMessage="No rules."
            />
            <EuiSpacer size="s" />
            <EuiFlexGroup gutterSize="s">
              <EuiFlexItem grow={2}>
                <EuiFieldText value={newReplace} onChange={(e) => setNewReplace(e.target.value)} placeholder="Regex replace" fullWidth />
              </EuiFlexItem>
              <EuiFlexItem grow={2}>
                <EuiFieldText value={newWith} onChange={(e) => setNewWith(e.target.value)} placeholder="With" fullWidth />
              </EuiFlexItem>
              <EuiFlexItem grow={false}>
                <EuiButton onClick={addReplacement} fill disabled={!newReplace.trim()}>Add rule</EuiButton>
              </EuiFlexItem>
            </EuiFlexGroup>
          </Fragment>
        );
      })()}
    </Fragment>
  );

  const INCLUSIONS_TABS = [
    { id: 'group_inclusions', label: 'Group inclusions', listKey: 'group_inclusions', color: 'success', desc: 'Keep only tracks whose group-title matches any of these patterns.' },
    { id: 'group_exclusions', label: 'Group exclusions', listKey: 'group_exclusions', color: 'danger', desc: 'Remove tracks whose group-title matches any of these patterns.' },
    { id: 'channel_inclusions', label: 'Channel inclusions', listKey: 'channel_inclusions', color: 'success', desc: 'Keep only tracks whose channel name matches any of these patterns.' },
    { id: 'channel_exclusions', label: 'Channel exclusions', listKey: 'channel_exclusions', color: 'danger', desc: 'Remove tracks whose channel name matches any of these patterns.' },
  ];

  const inclusionsContent = (
    <Fragment>
      <p className="euiTextColor--subdued" style={{ fontSize: '12px', marginBottom: 12 }}>
        Empty list = no filter (allow all for inclusions, exclude none for exclusions). Use the trash icon to remove a pattern.
      </p>
      {isMobile ? (
        // Four long tab labels don't fit a phone; a select keeps every section reachable.
        <EuiFormRow label="Section" fullWidth>
          <EuiSelect
            fullWidth
            options={patternSectionOptions}
            value={inclusionsSection}
            onChange={(e) => { setInclusionsSection(e.target.value); setActivePatternSection(e.target.value); }}
          />
        </EuiFormRow>
      ) : (
        <EuiTabs size="s">
          {INCLUSIONS_TABS.map((tab) => (
            <EuiTab key={tab.id} onClick={() => setInclusionsSection(tab.id)} isSelected={inclusionsSection === tab.id}>
              {tab.label}
            </EuiTab>
          ))}
        </EuiTabs>
      )}
      <EuiSpacer size="m" />
      {INCLUSIONS_TABS.map((tab) => inclusionsSection === tab.id && (
        <Fragment key={tab.id}>
          <EuiPanel paddingSize="m" color={tab.color} style={{ borderLeft: `4px solid ${tab.color === 'danger' ? '#bd271e' : '#017d73'}` }}>
            <EuiTitle size="xs"><h4 style={{ marginTop: 0 }}>{tab.label}</h4></EuiTitle>
            <p className="euiTextColor--subdued" style={{ fontSize: '12px' }}>{tab.desc}</p>
            <EuiSpacer size="s" />
            {renderPatternTable('', tab.listKey, '')}
          </EuiPanel>
          <EuiSpacer size="m" />
        </Fragment>
      ))}
      <EuiFlexGroup gutterSize="s" alignItems="flexEnd">
        <EuiFlexItem grow={false} style={{ minWidth: 180 }}>
          <EuiFormRow label="Add pattern to">
            <EuiSelect
              options={patternSectionOptions}
              value={activePatternSection}
              onChange={(e) => { setActivePatternSection(e.target.value); setInclusionsSection(e.target.value); }}
            />
          </EuiFormRow>
        </EuiFlexItem>
        <EuiFlexItem grow={2}>
          <EuiFormRow label="Pattern (regex)">
            <EuiFieldText
              value={newPattern}
              onChange={(e) => setNewPattern(e.target.value)}
              placeholder="e.g. ^Sports$ or channel name…"
              fullWidth
            />
          </EuiFormRow>
        </EuiFlexItem>
        <EuiFlexItem grow={false}>
          <EuiFormRow label=" ">
            <EuiButton onClick={() => addPattern(activePatternSection)} fill disabled={!newPattern.trim()}>Add</EuiButton>
          </EuiFormRow>
        </EuiFlexItem>
      </EuiFlexGroup>
    </Fragment>
  );

  return (
    <Fragment>
      <EuiPanel paddingSize="m" color="subdued">
        <EuiTitle size="xs"><h3>Processing order</h3></EuiTitle>
        <pre style={{ margin: '8px 0', fontFamily: 'monospace', fontSize: '13px', whiteSpace: 'pre-wrap' }}>{PROCESSING_DIAGRAM}</pre>
        <p className="euiTextColor--subdued" style={{ marginTop: 8 }}>
          <strong>1. Inclusions</strong> — Keep only tracks that match at least one pattern in group inclusions (if any) and at least one in channel inclusions (if any). Empty list = keep all.
          <br />
          <strong>2. Exclusions</strong> — Remove tracks whose group or channel name matches any exclusion pattern.
          <br />
          <strong>3. Replacements</strong> — Apply regex replace rules: global (names + groups), then names only, then groups only. Changes apply to the running proxy immediately (no restart needed).
        </p>
      </EuiPanel>
      <EuiSpacer size="l" />
      {message && message.includes('Error') && (
        <EuiCallOut title="Error" color="danger" iconType="alert">
          <p>{message}</p>
        </EuiCallOut>
      )}
      <EuiSpacer size="m" />

      <EuiTitle size="s"><h4>Configure processing</h4></EuiTitle>
      <EuiTabs size={isMobile ? 's' : 'm'}>
        <EuiTab onClick={() => setProcessingSubTab('replacements')} isSelected={processingSubTab === 'replacements'}>
          Replacements
        </EuiTab>
        <EuiTab onClick={() => setProcessingSubTab('inclusions')} isSelected={processingSubTab === 'inclusions'}>
          Inclusions &amp; exclusions
        </EuiTab>
      </EuiTabs>
      <EuiSpacer size="m" />
      {processingSubTab === 'replacements' && replacementsContent}
      {processingSubTab === 'inclusions' && inclusionsContent}

      <EuiSpacer size="l" />
      <EuiButton onClick={saveAll} fill isLoading={saving} isDisabled={loading} fullWidth={isMobile}>
        Save all processing settings
      </EuiButton>
    </Fragment>
  );
}

// --- Users Tab ---

function UsersTab({ addToast }) {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [modalMode, setModalMode] = useState(null); // 'add' | 'edit' | null
  const [editUser, setEditUser] = useState(null);
  const [deleteConfirm, setDeleteConfirm] = useState(null);

  const fetchUsers = useCallback(() => {
    setLoading(true);
    setError(null);
    fetch('/api/users')
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(r.statusText))))
      .then((data) => setUsers(data.users || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  const handleDelete = (username) => {
    fetch(`/api/users/${encodeURIComponent(username)}`, { method: 'DELETE' })
      .then((r) => {
        if (!r.ok) return r.json().then((d) => Promise.reject(new Error(d.error || r.statusText)));
        addToast(`User "${username}" deleted`);
        fetchUsers();
      })
      .catch((e) => addToast(`Error: ${e.message}`, 'danger'))
      .finally(() => setDeleteConfirm(null));
  };

  // Mobile card: username + status, description/created as secondary lines, touch-sized actions.
  const mobileColumn = {
    field: 'username',
    name: 'User',
    mobileOptions: {
      render: (item) => (
        <EuiFlexGroup gutterSize="s" alignItems="center" justifyContent="spaceBetween" responsive={false} wrap style={{ width: '100%' }}>
          <EuiFlexItem style={{ minWidth: 0 }}>
            <div style={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
              {item.username}
              <EuiBadge color={item.enabled ? 'success' : 'warning'} style={{ marginLeft: 6 }}>{item.enabled ? 'Enabled' : 'Disabled'}</EuiBadge>
            </div>
            {item.description && <div className="euiTextColor--subdued" style={{ fontSize: 12, overflowWrap: 'anywhere' }}>{item.description}</div>}
            {item.created_at && <div className="euiTextColor--subdued" style={{ fontSize: 12 }}>Created {item.created_at.split('T')[0]}</div>}
          </EuiFlexItem>
          <EuiFlexItem grow={false}>
            <TouchActions>
              <EuiButtonEmpty size="m" onClick={() => { setEditUser(item); setModalMode('edit'); }}>Edit</EuiButtonEmpty>
              <EuiButtonEmpty size="m" color="danger" onClick={() => setDeleteConfirm(item.username)}>Delete</EuiButtonEmpty>
            </TouchActions>
          </EuiFlexItem>
        </EuiFlexGroup>
      ),
    },
  };

  const columns = [
    { field: 'username', name: 'Username' },
    { field: 'description', name: 'Description', render: (val) => val || '—' },
    { field: 'enabled', name: 'Status', render: (val) => (
      <EuiBadge color={val ? 'success' : 'warning'}>{val ? 'Enabled' : 'Disabled'}</EuiBadge>
    )},
    { field: 'created_at', name: 'Created', render: (val) => val ? val.split('T')[0] : '—' },
    { name: 'Actions', render: (item) => (
      <EuiFlexGroup gutterSize="s" responsive={false}>
        <EuiFlexItem grow={false}>
          <EuiButtonEmpty size="xs" onClick={() => { setEditUser(item); setModalMode('edit'); }}>Edit</EuiButtonEmpty>
        </EuiFlexItem>
        <EuiFlexItem grow={false}>
          <EuiButtonEmpty size="xs" color="danger" onClick={() => setDeleteConfirm(item.username)}>Delete</EuiButtonEmpty>
        </EuiFlexItem>
      </EuiFlexGroup>
    )},
  ];

  return (
    <Fragment>
      <EuiFlexGroup justifyContent="spaceBetween" alignItems="center" responsive={false}>
        <EuiFlexItem grow={false}>
          <EuiTitle size="xs"><h3>Users ({users.length})</h3></EuiTitle>
        </EuiFlexItem>
        <EuiFlexItem grow={false}>
          <EuiButton size="s" fill onClick={() => { setEditUser(null); setModalMode('add'); }}>+ Add user</EuiButton>
        </EuiFlexItem>
      </EuiFlexGroup>
      <EuiSpacer size="m" />
      {error && <EuiCallOut title="Error" color="danger"><p>{error}</p></EuiCallOut>}
      <EuiBasicTable items={users} columns={withMobileSummary(mobileColumn, columns)} loading={loading} />

      {modalMode && (
        <UserFormModal
          mode={modalMode}
          user={editUser}
          onClose={() => { setModalMode(null); setEditUser(null); }}
          onSaved={() => { setModalMode(null); setEditUser(null); fetchUsers(); addToast(modalMode === 'add' ? 'User created' : 'User updated'); }}
        />
      )}

      {deleteConfirm && (
        <EuiConfirmModal
          title={`Delete user "${deleteConfirm}"?`}
          onCancel={() => setDeleteConfirm(null)}
          onConfirm={() => handleDelete(deleteConfirm)}
          cancelButtonText="Cancel"
          confirmButtonText="Delete"
          buttonColor="danger"
        >
          <p>This will permanently remove the user. Future connections using this account will fail, but existing sessions may continue until they reconnect.</p>
        </EuiConfirmModal>
      )}
    </Fragment>
  );
}

function UserFormModal({ mode, user, onClose, onSaved }) {
  const [username, setUsername] = useState(user?.username || '');
  const [password, setPassword] = useState('');
  const [description, setDescription] = useState(user?.description || '');
  const [enabled, setEnabled] = useState(user?.enabled ?? true);
  const [error, setError] = useState(null);
  const [saving, setSaving] = useState(false);
  const [loadingPassword, setLoadingPassword] = useState(false);

  const isEdit = mode === 'edit';

  // In edit mode, fetch the current password so the user can see/modify it.
  useEffect(() => {
    if (!isEdit || !user?.username) return;
    setLoadingPassword(true);
    fetch(`/api/users/${encodeURIComponent(user.username)}/watch`)
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => { if (data?.password) setPassword(data.password); })
      .catch(() => {})
      .finally(() => setLoadingPassword(false));
  }, [isEdit, user?.username]);

  const handleSave = () => {
    setError(null);
    if (!isEdit && !username.trim()) { setError('Username is required'); return; }
    if (!isEdit && !password) { setError('Password is required'); return; }
    if (!isEdit && !/^[a-zA-Z0-9_-]{1,64}$/.test(username)) { setError('Username must be 1-64 alphanumeric, hyphen, or underscore characters'); return; }

    setSaving(true);
    const url = isEdit ? `/api/users/${encodeURIComponent(user.username)}` : '/api/users';
    const method = isEdit ? 'PUT' : 'POST';
    const body = isEdit
      ? JSON.stringify({ password, description, enabled })
      : JSON.stringify({ username, password, description, enabled });

    fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body })
      .then((r) => {
        if (!r.ok) return r.json().then((d) => Promise.reject(new Error(d.error || r.statusText)));
        onSaved();
      })
      .catch((e) => setError(e.message))
      .finally(() => setSaving(false));
  };

  return (
    <EuiModal onClose={onClose} style={{ maxWidth: 450 }}>
      <EuiModalHeader>
        <EuiModalHeaderTitle>{isEdit ? `Edit user: ${user.username}` : 'Add user'}</EuiModalHeaderTitle>
      </EuiModalHeader>
      <EuiModalBody>
        {error && <><EuiCallOut title={error} color="danger" size="s" /><EuiSpacer size="m" /></>}
        <EuiFormRow label="Username">
          <EuiFieldText
            value={isEdit ? user.username : username}
            onChange={(e) => setUsername(e.target.value)}
            disabled={isEdit}
            placeholder="alphanumeric, hyphen, underscore"
          />
        </EuiFormRow>
        <EuiSpacer size="m" />
        <EuiFormRow label="Password">
          <EuiFieldPassword
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            type="dual"
            isLoading={loadingPassword}
          />
        </EuiFormRow>
        <EuiSpacer size="m" />
        <EuiFormRow label="Description" helpText="Optional note (e.g. device, person, purpose)">
          <EuiFieldText
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="e.g. Living room TV, Mom's phone"
          />
        </EuiFormRow>
        <EuiSpacer size="m" />
        <EuiSwitch label="Enabled" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
      </EuiModalBody>
      <EuiModalFooter>
        <EuiButtonEmpty onClick={onClose}>Cancel</EuiButtonEmpty>
        <EuiButton onClick={handleSave} fill isLoading={saving}>
          {isEdit ? 'Save changes' : 'Create user'}
        </EuiButton>
      </EuiModalFooter>
    </EuiModal>
  );
}
