const api = () => window.go.main.DesktopAPI;
const $ = (selector) => document.querySelector(selector);
const esc = (value) => String(value ?? "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[char]));
const platformNames = { feishu: "飞书", dingtalk: "钉钉", wecom: "企业微信" };

let state;
let activePage = "status";
let activeSetting = "detection";
let activeDeviceTab = "inventory";
let latestAssetChecks = [];
let cameraSelection = new Set();
let batchManaging = false;
let batchSelected = new Set();

function deviceComponentEnabled() {
  return state.components.some((item) => item.id === "device-status" && item.enabled);
}
function componentEnabled(id) {
  return state.components.some((item) => item.id === id && item.enabled);
}

function activePlatformState() {
  const name = platformNames[state.config.activePlatform] || "飞书";
  return state.platforms.find((item) => item.name === name) || { name, status: "未配置" };
}

function shell() {
  const deviceNav = deviceComponentEnabled() ? [["assets", "设备"]] : [];
  const navigation = [["status", "状态"], ...deviceNav, ["settings", "设置"], ["logs", "日志"]];
  return `
    <header class="top">
      <div class="brand">XinT-LanPulse</div>
      <nav class="nav" aria-label="主导航">
        ${navigation.map(([id, label]) => `<button type="button" data-page="${id}" class="${id === activePage ? "active" : ""}">${label}</button>`).join("")}
      </nav>
    </header>
    <main>
      <section class="page ${activePage === "status" ? "active" : ""}" id="page-status"></section>
      <section class="page ${activePage === "assets" ? "active" : ""}" id="page-assets"></section>
      <section class="page ${activePage === "settings" ? "active" : ""}" id="page-settings"></section>
      <section class="page ${activePage === "logs" ? "active" : ""}" id="page-logs"></section>
    </main>
    <div id="toast" role="status" aria-live="polite"></div>
    <div id="modal-root"></div>`;
}

function showToast(text, type = "success") {
  const toast = $("#toast");
  toast.textContent = text;
  toast.className = `show ${type}`;
  clearTimeout(showToast.timer);
  showToast.timer = setTimeout(() => { toast.className = ""; }, 3600);
}

async function call(action, successText) {
  try {
    const result = await action();
    if (successText) showToast(successText);
    return result;
  } catch (error) {
    showToast(error?.message || String(error), "error");
    return undefined;
  }
}

function renderStatus() {
  const config = state.config;
  const integration = activePlatformState();
  $("#page-status").innerHTML = `
    <div class="page-heading"><div><p class="eyebrow">RUNTIME</p><h1>状态</h1></div><span class="runtime">启动于 ${esc(state.startedAt)}</span></div>
    <div class="status-grid">
      <article class="status-card"><h2>接入状态</h2><strong id="connectionStatusValue">${esc(integration.status)}</strong><small>${esc(integration.name)}</small></article>
      <article class="status-card"><h2>本机设备</h2><strong>${esc(config.deviceName)}</strong><small>${esc(config.deviceID)}</small></article>
      <article class="status-card"><h2>最近检测</h2><strong id="latestPingValue">${esc(state.latestPing)}</strong><small>动态更新</small></article>
    </div>
    <section class="panel ping-panel"><div class="panel-heading"><div><h2>临时检测</h2><p>基础 Ping 功能。输入网段前缀和最后一位，检测不会重置正在填写的内容。</p></div></div>
      <div class="ping-row"><input id="pingPrefix" aria-label="IP 网段前缀" value="${esc(config.prefix)}"><input id="pingSuffix" aria-label="IP 最后一位" inputmode="numeric" placeholder="最后一位，例如 15"><button type="button" class="button primary" id="pingButton">立即检测</button></div><p class="result" id="pingResult"></p></section>`;
  const button = $("#pingButton");
  if (button) button.onclick = async () => {
    const prefix = $("#pingPrefix").value;
    const suffix = $("#pingSuffix").value;
    const result = $("#pingResult");
    button.disabled = true;
    result.textContent = "正在检测…";
    const text = await call(() => api().TestPingTarget(prefix, suffix));
    result.textContent = text || "检测未完成";
    button.disabled = false;
  };
}

function renderAssets() {
  if (!deviceComponentEnabled()) return;
  const cameraEnabled = componentEnabled("ipcamera-addon");
  if (activeDeviceTab === "camera" && !cameraEnabled) activeDeviceTab = "inventory";
  const tabs = [["inventory", "设备列表"], ["schedule", "定时任务"], ...(cameraEnabled ? [["camera", "监控管理"]] : [])];
  $("#page-assets").innerHTML = `<div class="page-heading"><div><p class="eyebrow">DEVICE WORKSPACE</p><h1>设备</h1></div></div><div class="device-tabs">${tabs.map(([id, label]) => `<button type="button" data-device-tab="${id}" class="${activeDeviceTab === id ? "active" : ""}">${label}</button>`).join("")}</div><div id="device-workspace"></div>`;
  document.querySelectorAll("[data-device-tab]").forEach((button) => { button.onclick = () => { activeDeviceTab = button.dataset.deviceTab; renderAssets(); }; });
  if (activeDeviceTab === "inventory") renderInventory();
  else if (activeDeviceTab === "schedule") renderDeviceSchedule();
  else renderCameraManagement();
}

function renderInventory() {
  const rows = state.assets.length ? state.assets.map((asset) => {
    const result = latestAssetChecks.find((item) => item.suffix === asset.suffix);
    const status = result ? `<span class="check-state ${result.online ? "online" : "offline"}">${result.online ? "在线" : "离线"}</span>` : `<span class="check-state">未检测</span>`;
    const selector = batchManaging ? `<input type="checkbox" data-batch-asset="${asset.suffix}" ${batchSelected.has(asset.suffix) ? "checked" : ""} aria-label="选择 ${esc(asset.name || asset.suffix)}" style="width:16px;height:16px;vertical-align:middle">` : "";
    const action = batchManaging ? "" : `<button class="text-button" data-edit-asset="${asset.suffix}">编辑</button>`;
    return `<tr><td style="text-align:center">${selector}</td><td style="text-align:center">${esc(state.config.prefix)}.${asset.suffix}</td><td style="text-align:center">${esc(asset.name || "未命名设备")}</td><td style="text-align:center">${esc(asset.area || "未设置区域")}</td><td style="text-align:center">${status}</td><td style="text-align:center">${action}</td></tr>`;
  }).join("") : `<tr><td colspan="6" class="table-empty">尚未导入设备。可通过“导入”或“添加设备”建立设备列表。</td></tr>`;
  const results = latestAssetChecks.length ? `<section class="panel batch-results"><div class="panel-heading split"><div><h2>本次批量检测结果</h2><p>${latestAssetChecks.filter((item) => item.online).length}/${latestAssetChecks.length} 台设备在线</p></div></div><div class="table-panel"><table><thead><tr><th>状态</th><th>设备</th><th>区域</th><th>IP</th><th>响应时间</th><th>检测次数</th><th>检测时间</th></tr></thead><tbody>${latestAssetChecks.map((item) => `<tr><td><span class="check-state ${item.online ? "online" : "offline"}">${item.online ? "可连接" : "无法连接"}</span></td><td>${esc(item.name || "未命名设备")}</td><td>${esc(item.area || "未设置区域")}</td><td>${esc(item.address)}</td><td>${item.online ? `${item.latencyMS} ms` : "-"}</td><td>${item.attempts}</td><td>${esc(item.checkedAt)}</td></tr>`).join("")}</tbody></table></div></section>` : "";
  const toolbar = batchManaging ? `<button class="button danger-button" id="deleteBatch">删除已选设备${batchSelected.size ? `（${batchSelected.size}）` : ""}</button><button class="button" id="cancelBatch">取消</button>` : `<button class="button" id="addAsset">添加设备</button><button class="button" id="importAssets">导入</button><button class="button primary" id="bulkCheck">批量检测</button><button class="button" id="manageBatch">批量管理</button>`;
  const allSelected = state.assets.length > 0 && batchSelected.size === state.assets.length;
  const selectAll = batchManaging ? `<input type="checkbox" id="selectAllBatch" ${allSelected ? "checked" : ""} aria-label="全选设备" style="width:16px;height:16px;vertical-align:middle">` : "";
  $("#device-workspace").innerHTML = `<div class="workspace-toolbar"><p class="workspace-note">维护设备台账，并对当前列表执行批量连通性检测。</p><div class="toolbar">${toolbar}</div></div><section class="panel table-panel"><table><thead><tr><th style="text-align:center">${selectAll}</th><th style="text-align:center">设备 IP</th><th style="text-align:center">设备名称</th><th style="text-align:center">设备区域</th><th style="text-align:center">状态</th><th style="text-align:center">操作</th></tr></thead><tbody>${rows}</tbody></table></section>${results}`;
  const add = $("#addAsset"); if (add) add.onclick = () => renderAssetForm();
  const importButton = $("#importAssets"); if (importButton) importButton.onclick = renderImportAssetsModal;
  const manage = $("#manageBatch"); if (manage) manage.onclick = () => { batchManaging = true; batchSelected = new Set(); renderInventory(); };
  const cancel = $("#cancelBatch"); if (cancel) cancel.onclick = () => { batchManaging = false; batchSelected = new Set(); renderInventory(); };
  const deleteBatch = $("#deleteBatch"); if (deleteBatch) deleteBatch.onclick = async () => {
    if (!batchSelected.size) { showToast("请先勾选要删除的设备", "error"); return; }
    if (!confirm(`删除已选择的 ${batchSelected.size} 台设备？`)) return;
    const done = await call(() => api().DeleteAssets(Array.from(batchSelected)));
    if (done !== undefined) { batchManaging = false; batchSelected = new Set(); latestAssetChecks = []; await reloadState(); renderInventory(); showToast("已删除选中设备"); }
  };
  const bulkCheck = $("#bulkCheck"); if (bulkCheck) bulkCheck.onclick = async () => {
    const button = $("#bulkCheck");
    button.disabled = true; button.textContent = "正在检测…";
    const items = await call(() => api().BulkCheckAssets());
    if (items !== undefined) { latestAssetChecks = items; renderInventory(); showToast(`批量检测完成，共 ${items.length} 台`); }
    else { button.disabled = false; button.textContent = "批量检测"; }
  };
  document.querySelectorAll("[data-batch-asset]").forEach((input) => { input.onchange = () => { const suffix = Number(input.dataset.batchAsset); if (input.checked) batchSelected.add(suffix); else batchSelected.delete(suffix); renderInventory(); }; });
  const selectAllBatch = $("#selectAllBatch"); if (selectAllBatch) selectAllBatch.onchange = () => { batchSelected = selectAllBatch.checked ? new Set(state.assets.map((asset) => asset.suffix)) : new Set(); renderInventory(); };
  document.querySelectorAll("[data-edit-asset]").forEach((button) => { button.onclick = () => { const item = state.assets.find((asset) => asset.suffix === Number(button.dataset.editAsset)); if (item) renderAssetForm(item); }; });
}

function renderAssetForm(asset) {
  const editing = Boolean(asset);
  const address = asset ? `${state.config.prefix}.${asset.suffix}` : state.config.prefix + ".";
  $("#modal-root").innerHTML = `<div class="modal-backdrop"><section class="modal" role="dialog" aria-modal="true" aria-label="${editing ? "设备编辑" : "添加设备"}"><button class="modal-close" id="closeAssetForm" aria-label="关闭">×</button><h2>${editing ? "设备编辑" : "添加设备"}</h2><p>设备 IP 必须位于当前检测网段 ${esc(state.config.prefix)}。</p><div class="form-grid" style="margin-top:18px"><label>设备 IP<input id="assetAddress" value="${esc(address)}" placeholder="${esc(state.config.prefix)}.15"></label><label>设备名称<input id="assetName" value="${esc(asset?.name || "")}" placeholder="例如：办公楼一楼门口"></label><label>设备区域<input id="assetArea" value="${esc(asset?.area || "")}" placeholder="例如：办公楼一楼"></label></div><div class="form-actions">${editing ? `<button class="button danger-button" id="deleteAssetFromEdit">删除设备</button>` : ""}<button class="button primary" id="saveAssetForm">${editing ? "保存修改" : "添加设备"}</button></div></section></div>`;
  $("#closeAssetForm").onclick = () => { $("#modal-root").innerHTML = ""; };
  $("#saveAssetForm").onclick = async () => {
    const input = { address: $("#assetAddress").value, name: $("#assetName").value, area: $("#assetArea").value };
    const done = await call(() => editing ? api().UpdateAsset(asset.suffix, input) : api().AddAsset(input));
    if (done !== undefined) { latestAssetChecks = []; await reloadState(); $("#modal-root").innerHTML = ""; renderInventory(); showToast(editing ? "设备已更新" : "设备已添加"); }
  };
  const deleteButton = $("#deleteAssetFromEdit"); if (deleteButton) deleteButton.onclick = async () => {
    if (!confirm(`删除 ${asset.name || address}？`)) return;
    const done = await call(() => api().DeleteAsset(asset.suffix));
    if (done !== undefined) { latestAssetChecks = []; await reloadState(); $("#modal-root").innerHTML = ""; renderInventory(); showToast("设备已删除"); }
  };
}

function renderImportAssetsModal() {
  $("#modal-root").innerHTML = `<div class="modal-backdrop"><section class="modal" role="dialog" aria-modal="true" aria-label="导入设备"><button class="modal-close" id="closeImportModal" aria-label="关闭">×</button><h2>导入设备</h2><p>选择设备 Excel 文件导入，或先下载模板后填写。</p><div class="modal-options"><button class="button primary" id="chooseAssetFile">选择文件</button><button class="button" id="downloadAssetTemplate">下载 Excel 模板</button></div></section></div>`;
  $("#closeImportModal").onclick = () => { $("#modal-root").innerHTML = ""; };
  $("#chooseAssetFile").onclick = async () => { const count = await call(() => api().ImportAssets()); if (count !== undefined) { latestAssetChecks = []; await reloadState(); $("#modal-root").innerHTML = ""; renderInventory(); showToast(`已导入 ${count} 台设备`); } };
  $("#downloadAssetTemplate").onclick = () => call(() => api().ExportAssetTemplate(), "Excel 模板已保存");
}

function renderDeviceSchedule() {
  const task = state.config.deviceTask || state.deviceTask || { enabled: false, intervalMinutes: 15 };
  $("#device-workspace").innerHTML = `<section class="settings-content device-setting"><div class="content-heading"><h2>定时任务</h2><p>启用后，客户端会按指定时间间隔对设备列表执行批量 Ping 检测，结果会写入运行日志。</p></div><div class="schedule-form"><label class="switch-label"><input type="checkbox" id="deviceTaskEnabled" ${task.enabled ? "checked" : ""}><span>启用设备状态定时检测</span></label><label>检测间隔（分钟）<input id="deviceTaskInterval" type="number" min="1" max="1440" value="${Number(task.intervalMinutes) || 15}"></label></div><div class="form-actions"><button class="button primary" id="saveDeviceTask">保存定时任务</button></div></section>`;
  $("#saveDeviceTask").onclick = async () => {
    const done = await call(() => api().SaveDeviceTask({ enabled: $("#deviceTaskEnabled").checked, intervalMinutes: Number($("#deviceTaskInterval").value) }));
    if (done !== undefined) { await reloadState(); renderDeviceSchedule(); showToast("定时任务已保存"); }
  };
}

function renderCameraManagement() {
  const chosen = new Set(state.config.cameraAssetSuffixes || []);
  const cameras = state.assets.filter((asset) => chosen.has(asset.suffix));
  const items = cameras.length ? cameras.map((asset) => `<tr><td>${esc(asset.name || "未命名设备")}</td><td>${esc(asset.area || "未设置区域")}</td><td>${esc(state.config.prefix)}.${asset.suffix}</td><td><span class="muted">等待画面源配置</span></td></tr>`).join("") : `<tr><td colspan="4" class="table-empty">尚未设置监控设备。请点击“管理监控点”从设备列表添加。</td></tr>`;
  $("#device-workspace").innerHTML = `<section class="camera-intro"><div><p class="eyebrow">IPCAMERA-ADDON</p><h2>监控管理</h2><p>此页由监控设备增量包提供，用于管理监控设备的画面源与异常检测任务。</p></div><div class="toolbar"><span class="addon-badge">增量包已启用</span><button class="button primary" id="manageCameras">管理监控点</button></div></section><section class="panel table-panel"><table><thead><tr><th>设备名称</th><th>监控区域</th><th>设备 IP</th><th>画面状态</th></tr></thead><tbody>${items}</tbody></table></section><section class="empty-state compact"><h2>画面异常检测待接入</h2><p>在为设备配置 RTSP 或 ONVIF 画面源后，<code>ipcamera-addon</code> 将承担黑屏、遮挡、冻结与画面丢失检测。</p></section>`;
  $("#manageCameras").onclick = renderCameraModal;
}

function renderCameraModal() {
  cameraSelection = new Set(state.config.cameraAssetSuffixes || []);
  drawCameraModal();
}
function drawCameraModal() {
  const available = state.assets.filter((asset) => !cameraSelection.has(asset.suffix));
  const selected = state.assets.filter((asset) => cameraSelection.has(asset.suffix));
  const deviceRow = (asset, checkedName) => `<label style="display:flex;gap:8px;align-items:center;padding:8px 3px;border-bottom:1px solid var(--line)"><input type="checkbox" data-${checkedName}="${asset.suffix}" style="width:16px;height:16px;flex:none"><span style="min-width:0;overflow-wrap:anywhere">${esc(asset.name || "未命名设备")} <small class="muted">${esc(state.config.prefix)}.${asset.suffix}</small></span></label>`;
  const listPanel = (title, rows, emptyText) => `<section class="panel" style="display:flex;height:100%;min-width:0;flex-direction:column;padding:12px"><h2 style="flex:none">${title}</h2><div style="min-height:0;flex:1;margin-top:8px;overflow-x:hidden;overflow-y:auto;padding-right:5px">${rows.length ? rows.join("") : `<p class="muted">${emptyText}</p>`}</div></section>`;
  $("#modal-root").innerHTML = `<div class="modal-backdrop"><section class="modal" style="display:flex;width:calc(100vw - 48px);max-width:1200px;height:calc(100vh - 48px);max-height:720px;flex-direction:column" role="dialog" aria-modal="true" aria-label="管理监控点"><button class="modal-close" id="closeCameraModal" aria-label="关闭">×</button><h2>管理监控点</h2><p>勾选设备后使用中间箭头加入或移出，保存后才会更新监控管理页。</p><div style="position:relative;display:grid;min-height:0;flex:1;grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:116px;margin-top:18px;overflow:hidden"><div style="min-width:0;min-height:0">${listPanel("设备列表", available.map((asset) => deviceRow(asset, "available-camera")), "没有可添加的设备")}</div><div style="min-width:0;min-height:0">${listPanel("已添加的监控点", selected.map((asset) => deviceRow(asset, "selected-camera")), "尚未添加监控点")}</div><div style="position:absolute;top:50%;left:50%;z-index:2;display:grid;width:104px;gap:10px;transform:translate(-50%,-50%)"><button class="button primary" id="moveToCamera" title="设为监控设备" style="box-shadow:0 0 0 5px var(--bg)">加入 →</button><button class="button" id="moveFromCamera" title="移出监控设备" style="box-shadow:0 0 0 5px var(--bg)">← 移出</button></div></div><div class="form-actions" style="flex:none"><button class="button primary" id="saveCameras">保存监控点</button></div></section></div>`;
  $("#closeCameraModal").onclick = () => { $("#modal-root").innerHTML = ""; };
  $("#moveToCamera").onclick = () => {
    const inputs = Array.from(document.querySelectorAll("[data-available-camera]:checked"));
    if (!inputs.length) { showToast("请先在左侧勾选设备", "error"); return; }
    inputs.forEach((input) => cameraSelection.add(Number(input.dataset.availableCamera)));
    drawCameraModal();
  };
  $("#moveFromCamera").onclick = () => {
    const inputs = Array.from(document.querySelectorAll("[data-selected-camera]:checked"));
    if (!inputs.length) { showToast("请先在右侧勾选监控点", "error"); return; }
    inputs.forEach((input) => cameraSelection.delete(Number(input.dataset.selectedCamera)));
    drawCameraModal();
  };
  $("#saveCameras").onclick = async () => { const done = await call(() => api().SaveCameraAssets(Array.from(cameraSelection))); if (done !== undefined) { await reloadState(); $("#modal-root").innerHTML = ""; renderCameraManagement(); showToast("监控点已保存"); } };
}

function settingMenu() {
  const items = [["detection", "检测设置"], ["platform", "平台接入"], ["components", "组件设置"], ["program", "程序设置"], ["backup", "配置备份与导入"]];
  return `<aside class="settings-menu" aria-label="设置分类">${items.map(([id, label]) => `<button type="button" data-setting="${id}" class="${activeSetting === id ? "active" : ""}">${label}</button>`).join("")}</aside>`;
}

function detectionSettings() {
  const config = state.config;
  return `<section class="settings-content"><div class="content-heading"><h2>检测设置</h2><p>用于本机临时检测与设备列表批量检测。</p></div><div class="form-grid">
    <label>IP 网段前缀<input id="networkPrefix" value="${esc(config.prefix)}" placeholder="例如 192.168.20"></label>
    <label>单次超时（ms）<input id="timeoutMS" type="number" min="1" value="${config.timeoutMS}"></label>
    <label>检测次数<input id="attempts" type="number" min="1" value="${config.attempts}"></label>
    <label>同时检测线程数<input id="concurrentWorkers" type="number" min="1" max="64" value="${config.concurrentWorkers || 8}"></label>
  </div><div class="form-actions"><button class="button primary" id="saveDetection">保存检测设置</button></div></section>`;
}

function platformSettings() {
  const config = state.config;
  const active = config.activePlatform || "feishu";
  const choices = Object.entries(platformNames).map(([id, name]) => `<button type="button" class="platform-choice ${id === active ? "selected" : ""}" data-platform="${id}"><span>${id === active ? "✓" : ""}</span>${name}</button>`).join("");
  let fields;
  if (active === "feishu") fields = `<label>App ID<input id="platformKey" value="${esc(config.appID)}" autocomplete="off"></label><label>App Secret<input id="platformSecret" type="password" autocomplete="new-password" placeholder="留空则保持已保存的密钥"></label><p class="field-help">启用飞书长连接后，机器人私聊消息可直接由本机处理。</p>`;
  else if (active === "dingtalk") fields = `<label>AppKey<input id="platformKey" value="${esc(config.dingTalkAppKey)}" autocomplete="off"></label><label>AppSecret<input id="platformSecret" type="password" autocomplete="new-password" placeholder="留空则保持已保存的密钥"></label><p class="field-help">钉钉凭据会加密保存在本机。消息通道将在后续接入后启用。</p>`;
  else fields = `<label>Corp ID<input id="platformKey" value="${esc(config.weComCorpID)}" autocomplete="off"></label><label>Secret<input id="platformSecret" type="password" autocomplete="new-password" placeholder="留空则保持已保存的密钥"></label><p class="field-help">企业微信凭据会加密保存在本机。消息通道将在后续接入后启用。</p>`;
  return `<section class="settings-content"><div class="content-heading"><h2>平台接入</h2><p>一次只选择一个接入平台，切换后状态页将显示该平台的连接状态。</p></div><div class="platform-choices">${choices}</div><div class="platform-fields">${fields}</div><div class="form-actions"><button class="button primary" id="savePlatform">保存平台设置</button></div></section>`;
}

function componentSettings() {
  const rows = state.components.map((item) => `<tr><td><code>${esc(item.id)}</code></td><td>${esc(item.name)}</td><td>${esc(item.description)}</td><td><button class="status-toggle ${item.enabled ? "on" : ""}" data-toggle-component="${esc(item.id)}">${item.enabled ? "已启用" : "已禁用"}</button></td><td>${item.id === "device-status" ? `<span class="muted">内置组件</span>` : `<button class="text-button danger" data-delete-component="${esc(item.id)}">删除</button>`}</td></tr>`).join("");
  return `<section class="settings-content"><div class="content-heading split"><div><h2>组件设置</h2><p>组件按场景启用。组件包仅导入描述清单，不会执行外部代码。</p></div><button class="button primary" id="openComponentModal">添加组件</button></div><div class="table-panel component-table"><table><thead><tr><th>ID</th><th>名称</th><th>组件描述</th><th>启用状态</th><th>操作</th></tr></thead><tbody>${rows}</tbody></table></div></section>`;
}

function programSettings() {
  const config = state.config;
  return `<section class="settings-content"><div class="content-heading"><h2>程序设置</h2><p>这些选项仅影响当前 Windows 客户端。</p></div><div class="checkbox-list">
    <label><input id="autoStart" type="checkbox" ${config.autoStart ? "checked" : ""}> 开机自动启动</label>
    <label><input id="startMinimized" type="checkbox" ${config.startMinimized ? "checked" : ""}> 启动时最小化</label>
    <label><input id="closeToTray" type="checkbox" ${config.closeToTray ? "checked" : ""}> 关闭窗口时最小化到托盘</label>
    <label><input id="saveRuntimeLogs" type="checkbox" ${config.saveRuntimeLogs ? "checked" : ""}> 保存运行日志</label>
  </div><div class="form-actions"><button class="button primary" id="saveProgram">保存程序设置</button></div></section>`;
}

function backupSettings() {
  return `<section class="settings-content"><div class="content-heading"><h2>配置备份与导入</h2><p>备份包含本机设置、设备列表与组件清单。导入会替换当前配置。</p></div><div class="backup-actions"><button class="button" id="exportBackup">选择位置并备份</button><button class="button danger-button" id="importBackup">选择备份文件并导入</button></div></section>`;
}

function renderSettings() {
  const views = { detection: detectionSettings, platform: platformSettings, components: componentSettings, program: programSettings, backup: backupSettings };
  $("#page-settings").innerHTML = `<div class="page-heading"><div><p class="eyebrow">CONFIGURATION</p><h1>设置</h1></div></div><div class="settings-layout">${settingMenu()}${views[activeSetting]()}</div>`;
  document.querySelectorAll("[data-setting]").forEach((button) => { button.onclick = () => { activeSetting = button.dataset.setting; renderSettings(); }; });
  bindSettingActions();
}

function settingsPayload(overrides = {}) {
  return { deviceName: state.config.deviceName, appID: state.config.appID, appSecret: "", prefix: state.config.prefix, timeoutMS: state.config.timeoutMS, attempts: state.config.attempts, concurrentWorkers: state.config.concurrentWorkers || 8, allowedUserIDs: state.config.allowedUserIDs || [], adminUserIDs: state.config.adminUserIDs || [], autoStart: state.config.autoStart, startMinimized: state.config.startMinimized, closeToTray: state.config.closeToTray, saveRuntimeLogs: state.config.saveRuntimeLogs, ...overrides };
}

function bindSettingActions() {
  const detection = $("#saveDetection");
  if (detection) detection.onclick = async () => {
    const done = await call(() => api().SaveSettings(settingsPayload({ prefix: $("#networkPrefix").value, timeoutMS: Number($("#timeoutMS").value), attempts: Number($("#attempts").value), concurrentWorkers: Number($("#concurrentWorkers").value) })));
    if (done !== undefined) { await reloadState(); showToast("检测设置已保存"); }
  };
  document.querySelectorAll("[data-platform]").forEach((button) => { button.onclick = async () => {
    const done = await call(() => api().SelectPlatform(button.dataset.platform));
    if (done !== undefined) { await reloadState(); renderSettings(); renderStatus(); }
  }; });
  const savePlatform = $("#savePlatform");
  if (savePlatform) savePlatform.onclick = async () => {
    const done = await call(() => api().SavePlatformSettings({ platform: state.config.activePlatform, appKey: $("#platformKey").value, secret: $("#platformSecret").value }));
    if (done !== undefined) { await reloadState(); renderSettings(); renderStatus(); showToast("平台设置已保存"); }
  };
  const saveProgram = $("#saveProgram");
  if (saveProgram) saveProgram.onclick = async () => {
    const done = await call(() => api().SaveSettings(settingsPayload({ autoStart: $("#autoStart").checked, startMinimized: $("#startMinimized").checked, closeToTray: $("#closeToTray").checked, saveRuntimeLogs: $("#saveRuntimeLogs").checked })));
    if (done !== undefined) { await reloadState(); showToast("程序设置已保存"); }
  };
  const openModal = $("#openComponentModal");
  if (openModal) openModal.onclick = renderComponentModal;
  document.querySelectorAll("[data-toggle-component]").forEach((button) => { button.onclick = async () => {
    const item = state.components.find((value) => value.id === button.dataset.toggleComponent);
    const done = await call(() => api().SetComponentEnabled(item.id, !item.enabled));
    if (done !== undefined) { await reloadState(); renderAll(); }
  }; });
  document.querySelectorAll("[data-delete-component]").forEach((button) => { button.onclick = async () => {
    if (!confirm("删除该组件？对应的组件设置将一并移除。")) return;
    const done = await call(() => api().DeleteComponent(button.dataset.deleteComponent));
    if (done !== undefined) { await reloadState(); renderSettings(); }
  }; });
  const exportBackup = $("#exportBackup");
  if (exportBackup) exportBackup.onclick = () => call(() => api().ExportBackup(), "配置备份已保存");
  const importBackup = $("#importBackup");
  if (importBackup) importBackup.onclick = async () => {
    if (!confirm("导入将替换当前设置，是否继续？")) return;
    const done = await call(() => api().ImportBackup());
    if (done !== undefined) { await reloadState(); renderAll(); showToast("配置已导入"); }
  };
}

function renderComponentModal() {
  $("#modal-root").innerHTML = `<div class="modal-backdrop"><section class="modal" role="dialog" aria-modal="true" aria-label="添加组件"><button class="modal-close" id="closeModal" aria-label="关闭">×</button><h2>添加组件</h2><p>请选择组件清单来源。清单只包含 ID、名称和描述，不会执行任何外部程序。</p><div class="modal-options"><button class="button primary" id="importPackage">组件包文件上传</button><button class="button" id="showURL">URL 添加</button></div><div id="componentURLField"></div></section></div>`;
  $("#closeModal").onclick = () => { $("#modal-root").innerHTML = ""; };
  $("#importPackage").onclick = async () => {
    const item = await call(() => api().ImportComponentPackage());
    if (item) { await reloadState(); $("#modal-root").innerHTML = ""; renderSettings(); showToast(`组件“${item.name}”已添加`); }
  };
  $("#showURL").onclick = () => {
    $("#componentURLField").innerHTML = `<label class="url-field">HTTPS 组件清单地址<input id="componentURL" type="url" placeholder="https://example.com/component.json"></label><button class="button primary" id="importURL">验证并添加</button>`;
    $("#importURL").onclick = async () => {
      const item = await call(() => api().AddComponentURL($("#componentURL").value));
      if (item) { await reloadState(); $("#modal-root").innerHTML = ""; renderSettings(); showToast(`组件“${item.name}”已添加`); }
    };
  };
}

function renderLogs() {
  $("#page-logs").innerHTML = `<div class="page-heading"><div><p class="eyebrow">EVENTS</p><h1>运行日志</h1></div></div><section class="panel log-panel"><pre>${esc((state.logs || []).join("\n") || "暂无运行日志")}</pre></section>`;
}

function bindNavigation() {
  document.querySelectorAll("[data-page]").forEach((button) => { button.onclick = () => { activePage = button.dataset.page; document.querySelectorAll("[data-page]").forEach((item) => item.classList.toggle("active", item === button)); document.querySelectorAll(".page").forEach((item) => item.classList.toggle("active", item.id === `page-${activePage}`)); }; });
}

function renderAll() {
  if (activePage === "assets" && !deviceComponentEnabled()) activePage = "status";
  $("#app").innerHTML = shell();
  renderStatus(); renderAssets(); renderSettings(); renderLogs(); bindNavigation();
}

async function reloadState() { state = await api().GetState(); }

async function refreshLive() {
  try {
    const live = await api().GetLiveStatus();
    const connection = $("#connectionStatusValue");
    const latest = $("#latestPingValue");
    if (connection && activePlatformState().name === "飞书") connection.textContent = live.connection || "未配置";
    if (latest) latest.textContent = live.latestPing;
  } catch (_) { /* A transient desktop bridge error should not disturb an active form. */ }
}

document.addEventListener("DOMContentLoaded", async () => {
  try {
    await reloadState();
    renderAll();
    setInterval(refreshLive, 2500);
  } catch (error) {
    $("#app").innerHTML = `<main><section class="empty-state"><h1>正在连接本地服务</h1><p>${esc(error?.message || error)}</p></section></main>`;
  }
});
