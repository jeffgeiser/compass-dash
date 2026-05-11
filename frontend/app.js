/* compass-dash — single-page app, vanilla JS, no framework */
'use strict';

// ── API helpers ────────────────────────────────────────────────────────────────

async function apiFetch(path, opts = {}) {
  const res = await fetch('/api' + path, {
    headers: { 'Content-Type': 'application/json', ...opts.headers },
    ...opts,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || res.statusText);
  }
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) return res.json();
  return res.text();
}

// ── Router ────────────────────────────────────────────────────────────────────

const screens = ['review', 'files', 'activity', 'config'];
let currentScreen = 'review';

function navigate(screen) {
  if (!screens.includes(screen)) screen = 'review';
  currentScreen = screen;

  document.querySelectorAll('.screen').forEach(el => el.classList.remove('active'));
  document.querySelectorAll('nav a').forEach(el => el.classList.remove('active'));

  const screenEl = document.getElementById('screen-' + screen);
  if (screenEl) screenEl.classList.add('active');

  const navLink = document.querySelector(`nav a[data-screen="${screen}"]`);
  if (navLink) navLink.classList.add('active');

  switch (screen) {
    case 'review':   loadReviewScreen(); break;
    case 'files':    loadFilesScreen();  break;
    case 'activity': loadActivityScreen(); break;
    case 'config':   loadConfigScreen(); break;
  }
}

document.querySelectorAll('nav a').forEach(link => {
  link.addEventListener('click', e => {
    e.preventDefault();
    navigate(link.dataset.screen);
  });
});

// ── State ─────────────────────────────────────────────────────────────────────

let pendingRefinements = [];
let selectedRefinement = null;
let currentFileContent = null;

// ── Review screen ─────────────────────────────────────────────────────────────

async function loadReviewScreen() {
  showListPane();
  await Promise.all([loadStats(), loadRefinements()]);
}

async function loadStats() {
  try {
    const stats = await apiFetch('/stats');
    const pd = stats.pending_count;
    const oldestEl = document.getElementById('stat-oldest');
    const pendingEl = document.getElementById('stat-pending');

    pendingEl.textContent = pd;
    pendingEl.className = 'stat-value' + (pd > 5 ? ' warn' : pd === 0 ? ' ok' : '');

    const badge = document.getElementById('nav-pending-badge');
    if (pd > 0) { badge.textContent = pd; badge.classList.remove('hidden'); }
    else { badge.classList.add('hidden'); }

    const oldest = stats.oldest_pending_age_days;
    oldestEl.textContent = oldest > 0 ? Math.round(oldest) : '–';
    oldestEl.className = 'stat-value' + (oldest > 14 ? ' warn' : '');

    document.getElementById('stat-accepted').textContent = stats.accepted_this_month;
    document.getElementById('stat-rejected').textContent = stats.rejected_this_month;

    if (stats.last_review_date) {
      const d = new Date(stats.last_review_date);
      const days = Math.floor((Date.now() - d.getTime()) / 86400000);
      const lastEl = document.getElementById('stat-last-review');
      lastEl.textContent = days === 0 ? 'today' : days === 1 ? 'yesterday' : days + 'd ago';
      lastEl.className = 'stat-value' + (days > 7 ? ' warn' : ' ok');
    } else {
      document.getElementById('stat-last-review').textContent = 'never';
    }
  } catch (err) {
    console.error('stats error:', err);
  }
}

async function loadRefinements() {
  const listEl = document.getElementById('refinement-list');
  listEl.innerHTML = '<div class="empty-state"><div class="spinner"></div></div>';
  try {
    pendingRefinements = await apiFetch('/refinements');
    renderRefinementList();
  } catch (err) {
    listEl.innerHTML = `<div class="notice error">${escHTML(err.message)}</div>`;
  }
}

function renderRefinementList() {
  const listEl = document.getElementById('refinement-list');
  if (!pendingRefinements.length) {
    listEl.innerHTML = `<div class="empty-state all-caught-up">
      <div class="empty-icon-circle">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
      </div>
      <h3>All caught up!</h3>
      <p>No pending refinements to review.</p>
    </div>`;
    return;
  }

  listEl.innerHTML = `<div class="refinement-grid">${pendingRefinements.map(r => `
    <div class="card" style="cursor:pointer" data-id="${escAttr(r.id)}">
      <div class="card-header">
        <h3 class="refinement-title">${escHTML((r.frontmatter && r.frontmatter.title) || r.id)}</h3>
      </div>
      <div class="card-meta">
        ${r.change_type ? `<span class="tag ${escAttr(r.change_type.toLowerCase())}">${escHTML(r.change_type)}</span>` : ''}
        ${r.confidence ? `<span class="tag ${escAttr(r.confidence.toLowerCase())}">${escHTML(r.confidence)} confidence</span>` : ''}
        ${r.proposed_by ? `<span class="tag">${escHTML(r.proposed_by)}</span>` : ''}
        ${r.proposed_at ? `<span class="text-sm text-muted">${formatRelativeDate(r.proposed_at)}</span>` : ''}
      </div>
      <p class="refinement-target">${escHTML(r.target_file || '')}${r.target_section ? ' › ' + escHTML(r.target_section) : ''}</p>
      ${r.observation_preview ? `<p class="refinement-observation">${escHTML(r.observation_preview)}</p>` : ''}
    </div>
  `).join('')}</div>`;

  listEl.querySelectorAll('.card[data-id]').forEach(el => {
    el.addEventListener('click', () => {
      const r = pendingRefinements.find(x => x.id === el.dataset.id);
      if (r) openRefinementDetail(r);
    });
  });
}

function openRefinementDetail(r) {
  selectedRefinement = r;
  currentFileContent = null;

  document.getElementById('detail-title').textContent = (r.frontmatter && r.frontmatter.title) || r.id;
  document.getElementById('detail-target').textContent =
    (r.target_file || '') + (r.target_section ? ' › ' + r.target_section : '');

  const meta = document.getElementById('detail-meta');
  meta.innerHTML = [
    r.change_type && `<span class="tag ${escAttr(r.change_type.toLowerCase())}">${escHTML(r.change_type)}</span>`,
    r.confidence && `<span class="tag ${escAttr(r.confidence.toLowerCase())}">${escHTML(r.confidence)} confidence</span>`,
    r.proposed_by && `<span class="tag">${escHTML(r.proposed_by)}</span>`,
    r.proposed_at && `<span class="text-sm text-muted">${formatRelativeDate(r.proposed_at)}</span>`,
  ].filter(Boolean).join('');

  const sections = document.getElementById('detail-sections');
  sections.innerHTML = [
    renderDetailSection('Observation', r.observation),
    renderDetailSection('Proposed change', r.proposed_change),
    renderDetailSection('Reasoning', r.reasoning),
    renderDetailSection('Evidence', r.evidence),
    r.suggested_follow_up ? renderDetailSection('Suggested follow-up', r.suggested_follow_up) : '',
  ].filter(Boolean).join('');

  hideEditPanel();
  hidePreviewPanel();

  showDetailPane();

  // Auto-load preview if target file is set
  if (r.target_file && r.change_type && r.proposed_change) {
    loadPreview(r);
  }
}

function renderDetailSection(title, content) {
  if (!content || !content.trim()) return '';
  return `<div class="detail-section card">
    <h3>${escHTML(title)}</h3>
    <div class="content">${markdownRender(content)}</div>
  </div>`;
}

async function loadPreview(r) {
  if (!r.target_file) return;
  showPreviewPanel();
  const previewEl = document.getElementById('preview-content');
  const errorEl = document.getElementById('preview-error');
  previewEl.textContent = 'Loading preview…';
  errorEl.classList.add('hidden');
  try {
    const result = await apiFetch('/refinements/' + encodeURIComponent(r.id) + '/preview', { method: 'POST' });
    previewEl.textContent = result.preview_content;
    currentFileContent = result.preview_content;
  } catch (err) {
    previewEl.textContent = '';
    errorEl.textContent = 'Preview unavailable: ' + err.message + ' — use Edit & Accept to apply manually.';
    errorEl.classList.remove('hidden');
    // Load raw file content for edit flow
    try {
      const raw = await apiFetch('/files/' + encodeURIComponent(r.target_file));
      currentFileContent = raw.content;
    } catch (_) { /* ok */ }
  }
}

// Accept: auto-apply the proposed change
document.getElementById('btn-accept').addEventListener('click', async () => {
  if (!selectedRefinement) return;
  try {
    await apiFetch('/refinements/' + encodeURIComponent(selectedRefinement.id) + '/accept', { method: 'POST' });
    showListPane();
    await loadReviewScreen();
  } catch (err) {
    alert('Failed to accept: ' + err.message);
  }
});

// Edit & Accept: load the file content into the textarea
document.getElementById('btn-edit-accept').addEventListener('click', async () => {
  if (!selectedRefinement) return;
  let content = currentFileContent;
  if (!content && selectedRefinement.target_file) {
    try {
      const raw = await apiFetch('/files/' + encodeURIComponent(selectedRefinement.target_file));
      content = raw.content;
    } catch (err) {
      alert('Could not load file: ' + err.message);
      return;
    }
  }
  document.getElementById('edit-textarea').value = content || '';
  showEditPanel();
});

// Accept with edits: write the edited content then move refinement
document.getElementById('btn-accept-edited').addEventListener('click', async () => {
  if (!selectedRefinement) return;
  const content = document.getElementById('edit-textarea').value;
  try {
    await apiFetch('/refinements/' + encodeURIComponent(selectedRefinement.id) + '/accept-edited', {
      method: 'POST',
      body: JSON.stringify({ content }),
    });
    showListPane();
    await loadReviewScreen();
  } catch (err) {
    alert('Failed to save: ' + err.message);
  }
});

document.getElementById('btn-cancel-edit').addEventListener('click', hideEditPanel);

// Reject flow
document.getElementById('btn-reject').addEventListener('click', () => {
  document.getElementById('reject-reason').value = '';
  document.getElementById('reject-modal').classList.add('open');
});

document.getElementById('btn-reject-cancel').addEventListener('click', () => {
  document.getElementById('reject-modal').classList.remove('open');
});

document.getElementById('btn-reject-confirm').addEventListener('click', async () => {
  if (!selectedRefinement) return;
  const reason = document.getElementById('reject-reason').value.trim();
  try {
    await apiFetch('/refinements/' + encodeURIComponent(selectedRefinement.id) + '/reject', {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
    document.getElementById('reject-modal').classList.remove('open');
    showListPane();
    await loadReviewScreen();
  } catch (err) {
    alert('Failed to reject: ' + err.message);
  }
});

document.getElementById('back-to-list').addEventListener('click', () => {
  showListPane();
});

function showListPane() {
  document.getElementById('review-list-pane').classList.remove('hidden');
  document.getElementById('review-detail-pane').classList.add('hidden');
  selectedRefinement = null;
}

function showDetailPane() {
  document.getElementById('review-list-pane').classList.add('hidden');
  document.getElementById('review-detail-pane').classList.remove('hidden');
}

function showPreviewPanel() {
  document.getElementById('preview-panel').classList.remove('hidden');
}

function hidePreviewPanel() {
  document.getElementById('preview-panel').classList.add('hidden');
}

function showEditPanel() {
  document.getElementById('edit-panel').classList.remove('hidden');
  document.getElementById('preview-panel').classList.add('hidden');
}

function hideEditPanel() {
  document.getElementById('edit-panel').classList.add('hidden');
}

// ── Files screen ──────────────────────────────────────────────────────────────

const FILE_ICON = `<svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`;
const FOLDER_ICON = `<svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`;

async function loadFilesScreen() {
  const treeEl = document.getElementById('file-tree');
  treeEl.innerHTML = '<div class="empty-state"><div class="spinner"></div></div>';
  document.getElementById('file-viewer-placeholder').classList.remove('hidden');
  document.getElementById('file-viewer').classList.add('hidden');

  try {
    const files = await apiFetch('/files');
    renderFileTree(files);
  } catch (err) {
    treeEl.innerHTML = `<div class="notice error">${escHTML(err.message)}</div>`;
  }
}

function renderFileTree(files) {
  const treeEl = document.getElementById('file-tree');
  if (!files || !files.length) {
    treeEl.innerHTML = '<p class="text-sm text-muted">No files found.</p>';
    return;
  }

  // Group by directory using the `dir` field from the API
  const byDir = {};
  const rootFiles = [];
  files.forEach(f => {
    if (!f.dir || f.dir === '.') {
      rootFiles.push(f);
    } else {
      if (!byDir[f.dir]) byDir[f.dir] = [];
      byDir[f.dir].push(f);
    }
  });

  let html = '';

  rootFiles.forEach(f => {
    html += `<div class="tree-root-file" data-path="${escAttr(f.path)}">${FILE_ICON}<span>${escHTML(f.filename)}</span></div>`;
  });

  Object.keys(byDir).sort().forEach(dir => {
    html += `<div class="tree-folder" data-dir="${escAttr(dir)}">
      <div class="tree-folder-header">
        <span class="tree-chevron">▾</span>
        ${FOLDER_ICON}
        <span>${escHTML(dir)}</span>
      </div>
      <div class="tree-folder-files">
        ${byDir[dir].map(f => `<div class="tree-file" data-path="${escAttr(f.path)}">${FILE_ICON}<span>${escHTML(f.filename)}</span></div>`).join('')}
      </div>
    </div>`;
  });

  treeEl.innerHTML = html || '<p class="text-sm text-muted">No files found.</p>';

  // Folder toggle
  treeEl.querySelectorAll('.tree-folder-header').forEach(header => {
    header.addEventListener('click', () => {
      header.closest('.tree-folder').classList.toggle('collapsed');
    });
  });

  // File click
  treeEl.querySelectorAll('[data-path]').forEach(el => {
    el.addEventListener('click', () => openFile(el.dataset.path, el));
  });
}

let activeFileEl = null;

async function openFile(path, triggerEl) {
  if (activeFileEl) activeFileEl.classList.remove('active');
  activeFileEl = triggerEl || null;
  if (activeFileEl) activeFileEl.classList.add('active');

  document.getElementById('file-viewer-placeholder').classList.add('hidden');
  const viewerEl = document.getElementById('file-viewer');
  viewerEl.classList.remove('hidden');
  document.getElementById('file-viewer-path').textContent = path;
  const contentEl = document.getElementById('file-viewer-content');
  contentEl.innerHTML = '<div class="spinner"></div>';
  try {
    const data = await apiFetch('/files/' + encodeURIComponent(path));
    contentEl.innerHTML = markdownRender(data.content);
  } catch (err) {
    contentEl.innerHTML = `<div class="notice error">${escHTML(err.message)}</div>`;
  }
}

// ── Activity screen ───────────────────────────────────────────────────────────

async function loadActivityScreen() {
  const logEl = document.getElementById('activity-log');
  logEl.innerHTML = '<div class="empty-state"><div class="spinner"></div></div>';
  try {
    const entries = await apiFetch('/activity');
    renderActivityLog(entries);
  } catch (err) {
    logEl.innerHTML = `<div class="notice error">${escHTML(err.message)}</div>`;
  }
}

function renderActivityLog(entries) {
  const logEl = document.getElementById('activity-log');
  if (!entries || !entries.length) {
    logEl.innerHTML = '<div class="empty-state"><h3>No activity yet</h3><p>Actions you take in compass-dash are recorded here.</p></div>';
    return;
  }
  const badgeClass = { proposed: 'proposed', accepted: 'accepted', rejected: 'rejected' };
  logEl.innerHTML = `<div class="card"><div>${entries.map(e => {
    const type = (e.event_type || '').toLowerCase();
    const cls = badgeClass[type] || '';
    return `<div class="log-entry">
      <span class="log-time">${escHTML(formatLogDate(e.timestamp))}</span>
      <span class="log-badge ${cls}">${escHTML(e.event_type || '')}</span>
      <span class="log-desc">${escHTML(e.description || '')}</span>
    </div>`;
  }).join('')}</div></div>`;
}

// ── Config screen ─────────────────────────────────────────────────────────────

async function loadConfigScreen() {
  try {
    const cfg = await apiFetch('/config');
    document.getElementById('cfg-compass-path').value = cfg.compass_path || '';
    document.getElementById('compass-path-label').textContent = cfg.compass_path
      ? shortPath(cfg.compass_path) : 'No compass path set';
    renderConnectPrompt(cfg.compass_path || '~/compass');
  } catch (err) {
    showConfigNotice('error', 'Could not load config: ' + err.message);
  }
}

function renderConnectPrompt(compassPath) {
  const prompt = `I have a personal context folder called a compass-md at ${compassPath}/.

Before we start, please read:
- ${compassPath}/CLAUDE.md — full instructions for working with this Compass
- ${compassPath}/self/voice.md — my communication style and tone
- ${compassPath}/self/preferences.md — my preferences
- ${compassPath}/self/facts.md — facts about me
- ${compassPath}/self/decisions.md — past decisions

Follow the instructions in CLAUDE.md. In particular:
- Let the Compass files inform how you write, recommend, and respond
- If you observe something worth capturing (a corrected preference, a new pattern, a stated perspective), propose a refinement to ${compassPath}/refinements/pending/ using the format in CLAUDE.md
- Do not edit Compass files directly — always use the pending queue
- Maximum 3 refinements per session`;

  document.getElementById('connect-prompt').textContent = prompt;
}

document.getElementById('btn-copy-prompt').addEventListener('click', async () => {
  const text = document.getElementById('connect-prompt').textContent;
  try {
    await navigator.clipboard.writeText(text);
    const btn = document.getElementById('btn-copy-prompt');
    btn.textContent = 'Copied!';
    setTimeout(() => {
      btn.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> Copy`;
    }, 2000);
  } catch (_) {
    alert('Copy failed — select the text manually.');
  }
});

document.getElementById('config-form').addEventListener('submit', async e => {
  e.preventDefault();
  const compassPath = document.getElementById('cfg-compass-path').value.trim();
  try {
    await apiFetch('/config', {
      method: 'PUT',
      body: JSON.stringify({ compass_path: compassPath }),
    });
    document.getElementById('compass-path-label').textContent = compassPath ? shortPath(compassPath) : '';
    showConfigNotice('success', 'Settings saved.');
    // Reload stats/refinements with new path
    if (currentScreen === 'review') await loadReviewScreen();
  } catch (err) {
    showConfigNotice('error', 'Could not save: ' + err.message);
  }
});

function showConfigNotice(type, msg) {
  const el = document.getElementById('config-notice');
  el.className = 'notice ' + type;
  el.textContent = msg;
  el.classList.remove('hidden');
  setTimeout(() => el.classList.add('hidden'), 4000);
}

// ── Utilities ─────────────────────────────────────────────────────────────────

function escHTML(s) {
  if (!s) return '';
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function escAttr(s) { return escHTML(s || ''); }

function formatRelativeDate(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  const days = Math.floor((Date.now() - d.getTime()) / 86400000);
  if (days === 0) return 'today';
  if (days === 1) return 'yesterday';
  if (days < 7) return days + 'd ago';
  return d.toLocaleDateString();
}

function formatLogDate(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function shortPath(p) {
  const home = p.startsWith('/') ? p.replace(/^\/Users\/[^/]+/, '~') : p;
  return home.length > 40 ? '…' + home.slice(-37) : home;
}

// ── Boot ──────────────────────────────────────────────────────────────────────

(async function boot() {
  try {
    const cfg = await apiFetch('/config');
    document.getElementById('compass-path-label').textContent = cfg.compass_path
      ? shortPath(cfg.compass_path) : 'No compass path set';
  } catch (_) { /* ok */ }

  navigate('review');
})();
