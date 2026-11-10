'use strict';

const $ = (id) => document.getElementById(id);
const state = { cfg: null, status: null, view: 'overview', timer: null };

const VIEW_TITLES = {
  overview: 'Обзор', smb: 'SMB-сервер', shares: 'Общие диски', users: 'Пользователи',
  stats: 'Статистика', logs: 'Логи', settings: 'Настройки',
};

async function api(path, opts = {}) {
  const res = await fetch(path, {
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  });
  if (res.status === 401) {
    showLogin('Сессия истекла, войдите заново');
    throw new Error('unauthorized');
  }
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  const ct = res.headers.get('content-type') || '';
  return ct.includes('json') ? res.json() : res.text();
}

function showLogin(msg) {
  $('login').classList.remove('hidden');
  $('login-msg').textContent = msg || '';
  clearInterval(state.timer);
}

async function doLogin(ev) {
  ev.preventDefault();
  const r = await fetch('/api/login', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user: $('login-user').value.trim(), pass: $('login-pass').value }),
  });
  const data = await r.json().catch(() => ({}));
  if (data.status === 'ok') {
    $('login').classList.add('hidden');
    startApp();
  } else {
    $('login-msg').textContent = 'Неверный логин или пароль';
  }
}

function show(view) {
  state.view = view;
  document.querySelectorAll('.nav').forEach((b) => b.classList.toggle('active', b.dataset.view === view));
  document.querySelectorAll('.view').forEach((s) => s.classList.toggle('active', s.id === `view-${view}`));
  $('title').textContent = VIEW_TITLES[view] || view;
  if (view === 'logs') loadLogs();
  if (view === 'settings' || view === 'smb' || view === 'shares' || view === 'users') fillForms();
  if (view !== 'logs') refresh();
}

function fmtBytes(n) {
  if (!n) return '0 B';
  const u = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0, v = n;
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return `${v.toFixed(v >= 100 ? 0 : 1)} ${u[i]}`;
}

function fmtUptime(sec) {
  const d = Math.floor(sec / 86400), h = Math.floor((sec % 86400) / 3600), m = Math.floor((sec % 3600) / 60);
  return d > 0 ? `${d}д ${h}ч` : h > 0 ? `${h}ч ${m}м` : `${m}м`;
}

function sparkline(el, series, max, color, unit) {
  const w = Math.max(el.clientWidth || 600, 200);
  const h = Math.max(el.clientHeight || 140, 80);
  el.setAttribute('viewBox', `0 0 ${w} ${h}`);
  const pad = { l: 34, r: 8, t: 10, b: 16 };
  if (!series.length) { el.innerHTML = ''; return; }
  const iw = w - pad.l - pad.r, ih = h - pad.t - pad.b;
  const step = iw / Math.max(series.length - 1, 1);
  const y = (v) => pad.t + ih - (Math.min(v, max) / max) * ih;
  const pts = series.map((v, i) => `${(pad.l + i * step).toFixed(1)},${y(v).toFixed(1)}`);
  const area = `${pad.l},${pad.t + ih} ${pts.join(' ')} ${pad.l + iw},${pad.t + ih}`;
  const grid = [0, 0.5, 1].map((f) => {
    const gy = pad.t + ih - f * ih;
    const label = (max * f).toFixed(max < 10 ? 1 : 0);
    return `<line class="grid" x1="${pad.l}" y1="${gy}" x2="${pad.l + iw}" y2="${gy}"/>` +
      `<text x="2" y="${gy + 3}">${label}</text>`;
  }).join('');
  const last = series[series.length - 1];
  el.innerHTML =
    `<defs><linearGradient id="g${el.id}" x1="0" y1="0" x2="0" y2="1">` +
    `<stop offset="0%" stop-color="${color}" stop-opacity=".35"/><stop offset="100%" stop-color="${color}" stop-opacity="0"/>` +
    `</linearGradient></defs>` + grid +
    `<polygon points="${area}" fill="url(#g${el.id})"/>` +
    `<polyline points="${pts.join(' ')}" fill="none" stroke="${color}" stroke-width="2"/>` +
    `<circle cx="${pad.l + (series.length - 1) * step}" cy="${y(last)}" r="3" fill="${color}"/>` +
    `<text x="${w - pad.r}" y="${pad.t + 9}" text-anchor="end">${last.toFixed(unit === '%' ? 0 : 1)}${unit || ''}</text>`;
}

async function refresh() {
  try {
    const st = await api('/api/status');
    state.status = st;
    const s = st.stats;
    $('ver').textContent = 'v' + st.version;
    const running = st.service === 'running';
    $('srvdot').className = 'dot ' + (running ? 'ok' : 'err');
    $('srvtext').textContent = running ? 'SMB работает' : 'SMB остановлен';

    $('ov-service').textContent = running ? 'работает' : 'остановлен';
    $('ov-ip').textContent = st.smb_ip + ':445';
    $('ov-share').textContent = '\\\\' + st.smb_ip + '\\' + st.share;
    $('ov-root').textContent = st.root;
    $('ov-sessions').textContent = s.sessions;
    $('ov-net').innerHTML = `<span class="rx">↓ ${s.rx_kbps.toFixed(0)}</span> <span class="tx">↑ ${s.tx_kbps.toFixed(0)}</span> <small>КБ/с</small>`;

    const hist = s.history || [];
    const loadMax = Math.max(1, ...hist.map((p) => p.load1)) * 1.2;
    const netMax = Math.max(64, ...hist.map((p) => Math.max(p.rx_kbps, p.tx_kbps))) * 1.2;
    sparkline($('chart-load'), hist.map((p) => p.load1), loadMax, '#4ea1ff', '');
    sparkline($('chart-net'), hist.map((p) => p.rx_kbps), netMax, '#3ddc97', '');
    sparkline($('chart-load2'), hist.map((p) => p.load1), Math.max(1, ...hist.map((p) => p.load1)) * 1.2, '#7b5cff', '');

    $('st-uptime').textContent = fmtUptime(s.uptime_sec);
    $('st-load').textContent = s.load.map((v) => v.toFixed(2)).join(' / ');
    $('st-mem').textContent = `${s.mem_used_pc.toFixed(0)}%`;
    $('st-membar').style.width = s.mem_used_pc + '%';
    $('st-sessions').textContent = s.sessions;

    const shorten = (p) => p.replace(/^\/opt\/var\/keenetic-smb-ui\/share\/?/, '') || 'all';
    $('ov-disks').innerHTML = (s.disks || []).map((d) => {
      const cls = d.used_pc > 90 ? 'err' : d.used_pc > 75 ? 'warn' : 'ok';
      return `<tr><td><span class="diskname" title="${d.path}">${shorten(d.path)}</span></td>` +
        `<td><div class="diskrow"><div class="diskbar"><i style="width:${d.used_pc}%"></i></div>` +
        `<span>${fmtBytes(d.used)} · ${d.used_pc}%</span></div></td>` +
        `<td>${fmtBytes(d.free)}</td>` +
        `<td><span class="pill ${cls}">${d.used_pc}%</span></td></tr>`;
    }).join('') || '<tr><td colspan="4" class="hint">нет данных</td></tr>';

    if (state.view === 'shares') renderRemoteStatus(st.remotes);
  } catch (e) {
    $('srvdot').className = 'dot err';
    $('srvtext').textContent = 'нет связи с API';
  }
}

function renderRemoteStatus(list) {
  const map = new Map((list || []).map((r) => [r.id, r]));
  document.querySelectorAll('#remotes tr').forEach((tr) => {
    const st = map.get(tr.dataset.id);
    const cell = tr.querySelector('.status');
    if (!cell) return;
    if (!st) { cell.innerHTML = '<span class="pill">—</span>'; return; }
    cell.innerHTML = st.mounted ? '<span class="pill ok">подключён</span>' : '<span class="pill warn">не смонтирован</span>';
  });
}

function rowRemote(r = {}) {
  const tr = document.createElement('tr');
  tr.dataset.id = r.id || '';
  tr.innerHTML = `
    <td><input class="f-name" value="${r.name || ''}" placeholder="proxmox-24tb"></td>
    <td><input class="f-host" value="${r.host || ''}" placeholder="192.168.165.200"></td>
    <td><input class="f-share" value="${r.share || ''}" placeholder="hddshare24tb-2"></td>
    <td><input class="f-user" value="${r.user || ''}" placeholder="root"></td>
    <td><input class="f-pass" type="password" placeholder="пароль"></td>
    <td><input class="f-en" type="checkbox" ${r.enabled !== false ? 'checked' : ''}></td>
    <td class="status"><span class="pill">—</span></td>
    <td><button class="icon-btn" title="удалить">✕</button></td>`;
  tr.querySelector('.icon-btn').onclick = () => tr.remove();
  return tr;
}

function collectRemotes() {
  return [...document.querySelectorAll('#remotes tr')].map((tr) => ({
    id: tr.dataset.id,
    name: tr.querySelector('.f-name').value.trim(),
    host: tr.querySelector('.f-host').value.trim(),
    share: tr.querySelector('.f-share').value.trim(),
    user: tr.querySelector('.f-user').value.trim(),
    password: tr.querySelector('.f-pass').value,
    enabled: tr.querySelector('.f-en').checked,
  }));
}

function rowUser(u = {}) {
  const tr = document.createElement('tr');
  tr.dataset.id = u.name || '';
  tr.innerHTML = `
    <td><input class="f-name" value="${u.name || ''}" placeholder="admin"></td>
    <td><input class="f-pass" type="password" placeholder="без изменений"></td>
    <td><input class="f-ro" type="checkbox" ${u.readonly ? 'checked' : ''}></td>
    <td><input class="f-en" type="checkbox" ${u.enabled !== false ? 'checked' : ''}></td>
    <td><button class="icon-btn" title="удалить">✕</button></td>`;
  tr.querySelector('.icon-btn').onclick = () => tr.remove();
  return tr;
}

function collectUsers() {
  return [...document.querySelectorAll('#users tr')].map((tr) => ({
    name: tr.querySelector('.f-name').value.trim(),
    password: tr.querySelector('.f-pass').value,
    readonly: tr.querySelector('.f-ro').checked,
    enabled: tr.querySelector('.f-en').checked,
  }));
}

async function fillForms() {
  if (!state.cfg) state.cfg = await api('/api/config');
  const c = state.cfg;
  if (state.view === 'smb') {
    $('smb-ip').value = c.smb.listen_ip;
    $('smb-dev').value = c.smb.bind_device;
    $('smb-wg').value = c.smb.workgroup;
    $('smb-name').value = c.smb.server_string;
    $('smb-share').value = c.smb.share_name;
    $('smb-min').value = c.smb.min_protocol;
    $('smb-max').value = c.smb.max_protocol;
    $('smb-enabled').checked = !!c.smb.enabled;
    $('smb-guest').checked = !!c.smb.guest_ok;
  }
  if (state.view === 'shares') {
    const body = $('remotes');
    body.innerHTML = '';
    (c.remotes || []).forEach((r) => body.appendChild(rowRemote(r)));
  }
  if (state.view === 'users') {
    const body = $('users');
    body.innerHTML = '';
    (c.smb.users || []).forEach((u) => body.appendChild(rowUser(u)));
  }
  if (state.view === 'settings') {
    $('web-listen').value = c.web.listen;
    $('web-user').value = c.web.user;
    $('web-ro').checked = !!c.web.readonly;
    $('st-interval').value = c.stats.interval_sec;
    $('st-len').value = c.stats.history_len;
  }
}

async function saveConfig() {
  const cfg = state.cfg || (await api('/api/config'));
  cfg.smb.listen_ip = $('smb-ip').value.trim();
  cfg.smb.bind_device = $('smb-dev').value.trim();
  cfg.smb.workgroup = $('smb-wg').value.trim();
  cfg.smb.server_string = $('smb-name').value.trim();
  cfg.smb.share_name = $('smb-share').value.trim();
  cfg.smb.min_protocol = $('smb-min').value;
  cfg.smb.max_protocol = $('smb-max').value;
  cfg.smb.enabled = $('smb-enabled').checked;
  cfg.smb.guest_ok = $('smb-guest').checked;
  cfg.web.listen = $('web-listen').value.trim();
  cfg.web.user = $('web-user').value.trim();
  const p = $('web-pass').value;
  if (p) cfg.web.password = p;
  cfg.web.readonly = $('web-ro').checked;
  cfg.stats.interval_sec = Number($('st-interval').value) || 5;
  cfg.stats.history_len = Number($('st-len').value) || 120;
  await api('/api/config', { method: 'POST', body: JSON.stringify(cfg) });
  state.cfg = cfg;
  $('cfg-msg').textContent = 'Сохранено.';
}

async function startApp() {
  document.querySelectorAll('.nav').forEach((b) => (b.onclick = () => show(b.dataset.view)));
  $('btn-logout').onclick = async () => {
    try { await fetch('/api/logout', { method: 'POST', credentials: 'same-origin' }); } catch (e) { /* ignore */ }
    location.reload();
  };
  $('btn-refresh').onclick = refresh;
  $('btn-apply').onclick = async () => {
    try { const r = await api('/api/remotes/apply', { method: 'POST' }); console.log(r); } catch (e) { alert(e.message); }
    refresh();
  };
  $('remote-add').onclick = () => $('remotes').appendChild(rowRemote());
  $('user-add').onclick = () => $('users').appendChild(rowUser());
  $('users-import').onclick = async () => {
    const msg = $('users-msg');
    try {
      const r = await api('/api/ndms/users');
      if (!r.available) { msg.textContent = 'NDMS CLI недоступен на этом устройстве'; return; }
      if (r.error) { msg.textContent = 'Ошибка NDMS: ' + r.error; return; }
      const have = new Set([...document.querySelectorAll('#users tr .f-name')].map((i) => i.value.trim()));
      let added = 0;
      for (const u of r.users || []) {
        if (have.has(u.name)) continue;
        $('users').appendChild(rowUser({ name: u.name, enabled: true, readonly: !!u.readonly }));
        added++;
      }
      msg.textContent = added
        ? `Добавлено из роутера: ${added}. Задайте пароли и нажмите «Сохранить».`
        : 'Все пользователи роутера уже добавлены';
    } catch (e) {
      msg.textContent = 'Ошибка: ' + e.message;
    }
  };
  $('remotes-save').onclick = async () => {
    await api('/api/remotes', { method: 'POST', body: JSON.stringify({ remotes: collectRemotes() }) });
    state.cfg = null; fillForms(); refresh();
  };
  $('users-save').onclick = async () => {
    await api('/api/users', { method: 'POST', body: JSON.stringify(collectUsers()) });
    state.cfg = null; fillForms(); refresh();
  };
  $('cfg-save').onclick = async () => {
    try { await saveConfig(); await api('/api/remotes/apply', { method: 'POST' }); refresh(); }
    catch (e) { $('cfg-msg').textContent = 'Ошибка: ' + e.message; }
  };
  $('upd-check').onclick = async () => {
    const i = await api('/api/update');
    $('upd-current').textContent = i.current || '—';
    $('upd-msg').textContent = i.error
      ? 'Ошибка: ' + i.error
      : i.newer
        ? `Доступна ${i.latest} (сейчас ${i.current}) — нажмите «Установить»`
        : `Обновлений нет (последняя ${i.latest || i.current})`;
  };
  $('upd-install').onclick = async () => {
    $('upd-msg').textContent = 'Установка…';
    const r = await api('/api/update', { method: 'POST' });
    $('upd-msg').textContent = r.status === 'installing'
      ? 'Устанавливается, страница перезагрузится через 20 секунд…'
      : r.status === 'uptodate'
        ? 'Обновлений нет'
        : 'Ошибка: ' + (r.error || 'неизвестно');
    if (r.status === 'installing') setTimeout(() => location.reload(), 20000);
  };
  for (const [id, action] of [['smb-start', 'start'], ['smb-stop', 'stop'], ['smb-restart', 'restart']]) {
    $(id).onclick = async () => {
      const r = await api(`/api/service?action=${action}`, { method: 'POST' });
      console.log(r); refresh();
    };
  }
  $('smb-preview').onclick = async () => {
    const txt = await api('/api/config/raw');
    $('smb-conf').textContent = txt;
    $('smb-conf').classList.toggle('hidden');
  };
  $('btn-apply').onclick = async () => {
    try { await api('/api/remotes/apply', { method: 'POST' }); } catch (e) { /* показать ниже */ }
    refresh();
  };
  await refresh();
  state.timer = setInterval(refresh, 5000);
}

async function loadLogs() {
  try {
    $('logs').textContent = await api('/api/logs');
  } catch (e) {
    $('logs').textContent = 'Лог недоступен: ' + e.message;
  }
}

async function boot() {
  $('login-form').addEventListener('submit', doLogin);
  try {
    const s = await fetch('/api/session', { credentials: 'same-origin' }).then((r) => r.json());
    if (s.authenticated) {
      $('login').classList.add('hidden');
      startApp();
    } else {
      showLogin();
    }
  } catch (e) {
    showLogin('Панель недоступна');
  }
}

boot();
