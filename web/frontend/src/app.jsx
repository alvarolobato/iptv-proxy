import {
  EuiBadge,
  EuiBasicTable,
  EuiButton,
  EuiButtonEmpty,
  EuiButtonIcon,
  EuiCallOut,
  EuiFieldSearch,
  EuiFieldText,
  EuiFilterButton,
  EuiFilterGroup,
  EuiFlexGroup,
  EuiFlexItem,
  EuiFormRow,
  EuiIcon,
  EuiPageTemplate,
  EuiPanel,
  EuiSelect,
  EuiSpacer,
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
        maxWidth: 360,
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

const headerTitle = (
  <Link to="/" style={{ color: 'inherit', textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: 8 }}>
    <img src="/logo-128.png" alt="" width={32} height={32} style={{ display: 'block' }} />
    <span>IPTV-Proxy configuration</span>
  </Link>
);

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
  const isSettings = location.pathname === '/settings';

  const rightSideItems = (
    <EuiFlexGroup key="header-right" alignItems="center" gutterSize="m">
      <EuiFlexItem grow={false}>
        <EuiButton iconType="gear" onClick={() => navigate('/settings')} size="s">
          Settings
        </EuiButton>
      </EuiFlexItem>
    </EuiFlexGroup>
  );

  return (
    <EuiPageTemplate panelled={true}>
      <EuiPageTemplate.Header pageTitle={headerTitle} rightSideItems={[rightSideItems]} />
      <EuiPageTemplate.Section restrictWidth={1400}>
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

  const tabs = [
    { id: 'groups', name: 'Groups' },
    { id: 'channels', name: 'Channels' },
    { id: 'processing', name: 'Processing' },
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
    <EuiPanel paddingSize="l">
      <EuiTitle size="m">
        <h2>Data &amp; processing</h2>
      </EuiTitle>
      <EuiSpacer size="m" />
      <EuiTabs>
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
          addInProgress={processingAddInProgress}
          refreshKey={channelsRefreshKey}
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
      <ToastList toasts={toasts} />
    </EuiPanel>
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
                <EuiFlexGroup gutterSize="xs" alignItems="center">
                  <EuiFlexItem grow={true}>
                    <EuiFieldText
                      fullWidth
                      value={editingGroupValue}
                      onChange={(e) => setEditingGroupValue(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') onSaveGroupReplacement?.(editingGroupName, editingGroupValue)?.then(() => setEditingGroupName(null))?.catch(() => {});
                        if (e.key === 'Escape') setEditingGroupName(null);
                      }}
                      autoFocus
                    />
                  </EuiFlexItem>
                  <EuiFlexItem grow={false}>
                    <EuiToolTip content="Save">
                      <EuiButtonEmpty size="xs" iconType="check" color="primary" onClick={() => { onSaveGroupReplacement?.(editingGroupName, editingGroupValue)?.then(() => setEditingGroupName(null))?.catch(() => {}); }} aria-label="Save" />
                    </EuiToolTip>
                  </EuiFlexItem>
                  <EuiFlexItem grow={false}>
                    <EuiToolTip content="Cancel">
                      <EuiButtonEmpty size="xs" iconType="cross" color="danger" onClick={() => { setEditingGroupName(null); addToast?.('Canceled'); }} aria-label="Cancel" />
                    </EuiToolTip>
                  </EuiFlexItem>
                </EuiFlexGroup>
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
      <EuiBasicTable
        items={paginated}
        columns={columns}
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

function ChannelsTab({ groupFilter, showIncluded, showExcluded, onShowIncludedChange, onShowExcludedChange, onClearGroupFilter, onAddToProcessing, addInProgress, refreshKey }) {
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
        return (
          <EuiFlexGroup gutterSize="xs" alignItems="center" wrap>
            <EuiFlexItem grow={false}>
              <Fragment>
                {r.name ?? name ?? '—'}
                {r.tvg_name && r.tvg_name !== (r.name ?? name) && (
                  <Fragment>
                    <br />
                    <span className="euiTextColor--subdued" style={{ fontSize: '12px' }}>tvg-name: {r.tvg_name}</span>
                  </Fragment>
                )}
              </Fragment>
            </EuiFlexItem>
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
      width: '120px',
      render: (val, item) => {
        const row = getChannelRow(val, item);
        const channelName = row.name ?? '';
        return (
          <div style={{ display: 'inline-flex', alignItems: 'center', gap: 0 }}>
            <EuiToolTip content="Add to inclusions">
              <EuiButtonEmpty iconType="plusInCircleFilled" size="xs" color="success" onClick={() => onAddToProcessing({ section: 'channel_inclusions', value: channelName })} aria-label="Add to inclusions" isDisabled={addInProgress} />
            </EuiToolTip>
            <EuiToolTip content="Add to exclusions">
              <EuiButtonEmpty iconType="minusInCircleFilled" size="xs" color="danger" onClick={() => onAddToProcessing({ section: 'channel_exclusions', value: channelName })} aria-label="Add to exclusions" isDisabled={addInProgress} />
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
      <EuiBasicTable
        items={paginated}
        columns={columns}
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
          columns={[
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
          ]}
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
            {id === 'global' ? 'Global' : id === 'names' ? 'Names' : 'Groups'}
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
        return (
          <Fragment>
            <EuiBasicTable
              items={itemsWithIdx}
              columns={[
                {
                  field: 'replace',
                  name: 'Replace (regex)',
                  render: (val, row) => {
                    const idx = row != null && typeof row._idx === 'number' ? row._idx : -1;
                    return (
                    <EuiFlexGroup gutterSize="xs" alignItems="center" wrap>
                      <EuiFlexItem grow={true}>
                        {isEditing(idx, 'replace') ? (
                          <EuiFlexGroup gutterSize="xs" alignItems="center">
                            <EuiFlexItem grow={true}>
                              <EuiFieldText
                                fullWidth
                                value={replacementEditValue}
                                onChange={(e) => setReplacementEditValue(e.target.value)}
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter') applyReplacementEdit();
                                  if (e.key === 'Escape') cancelReplacementEdit();
                                }}
                                autoFocus
                              />
                            </EuiFlexItem>
                            <EuiFlexItem grow={false}>
                              <EuiToolTip content="Save">
                                <EuiButtonEmpty size="xs" iconType="check" color="primary" onClick={applyReplacementEdit} aria-label="Save" />
                              </EuiToolTip>
                            </EuiFlexItem>
                            <EuiFlexItem grow={false}>
                              <EuiToolTip content="Cancel">
                                <EuiButtonEmpty size="xs" iconType="cross" onClick={cancelReplacementEdit} aria-label="Cancel" />
                              </EuiToolTip>
                            </EuiFlexItem>
                          </EuiFlexGroup>
                        ) : (
                          <span>{val ?? ''}</span>
                        )}
                      </EuiFlexItem>
                      {!isEditing(idx, 'replace') && (
                        <EuiFlexItem grow={false}>
                          <EuiToolTip content="Edit">
                            <EuiButtonEmpty
                              size="xs"
                              iconType="pencil"
                              onClick={() => startReplacementEdit(key, idx, 'replace', val)}
                              aria-label="Edit replace pattern"
                            />
                          </EuiToolTip>
                        </EuiFlexItem>
                      )}
                    </EuiFlexGroup>
                    );
                  },
                },
                {
                  field: 'with',
                  name: 'With',
                  render: (val, row) => {
                    const idx = row != null && typeof row._idx === 'number' ? row._idx : -1;
                    return (
                    <EuiFlexGroup gutterSize="xs" alignItems="center" wrap>
                      <EuiFlexItem grow={true}>
                        {isEditing(idx, 'with') ? (
                          <EuiFlexGroup gutterSize="xs" alignItems="center">
                            <EuiFlexItem grow={true}>
                              <EuiFieldText
                                fullWidth
                                value={replacementEditValue}
                                onChange={(e) => setReplacementEditValue(e.target.value)}
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter') applyReplacementEdit();
                                  if (e.key === 'Escape') cancelReplacementEdit();
                                }}
                                autoFocus
                              />
                            </EuiFlexItem>
                            <EuiFlexItem grow={false}>
                              <EuiToolTip content="Save">
                                <EuiButtonEmpty size="xs" iconType="check" color="primary" onClick={applyReplacementEdit} aria-label="Save" />
                              </EuiToolTip>
                            </EuiFlexItem>
                            <EuiFlexItem grow={false}>
                              <EuiToolTip content="Cancel">
                                <EuiButtonEmpty size="xs" iconType="cross" onClick={cancelReplacementEdit} aria-label="Cancel" />
                              </EuiToolTip>
                            </EuiFlexItem>
                          </EuiFlexGroup>
                        ) : (
                          <span>{val ?? ''}</span>
                        )}
                      </EuiFlexItem>
                      {!isEditing(idx, 'with') && (
                        <EuiFlexItem grow={false}>
                          <EuiToolTip content="Edit">
                            <EuiButtonEmpty
                              size="xs"
                              iconType="pencil"
                              onClick={() => startReplacementEdit(key, idx, 'with', val)}
                              aria-label="Edit with value"
                            />
                          </EuiToolTip>
                        </EuiFlexItem>
                      )}
                    </EuiFlexGroup>
                    );
                  },
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
              ]}
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
      <EuiTabs size="s">
        {INCLUSIONS_TABS.map((tab) => (
          <EuiTab key={tab.id} onClick={() => setInclusionsSection(tab.id)} isSelected={inclusionsSection === tab.id}>
            {tab.label}
          </EuiTab>
        ))}
      </EuiTabs>
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
        <pre style={{ margin: '8px 0', fontFamily: 'monospace', fontSize: '13px' }}>{PROCESSING_DIAGRAM}</pre>
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
      <EuiTabs>
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
      <EuiButton onClick={saveAll} fill isLoading={saving} isDisabled={loading}>
        Save all processing settings
      </EuiButton>
    </Fragment>
  );
}
