"use strict";
const $ = id => document.getElementById(id);
const key = "lanpulse.devices";
let devices = [];
function saveDevices(next) {
  try { localStorage.setItem(key, JSON.stringify(next)); devices = next; renderDevices(); return true; }
  catch (_) { $("notice").textContent = "无法保存设备清单，请检查本地存储是否可用。"; return false; }
}
function validAddress(s) {
  const parts = s.split(".");
  return parts.length === 4 && parts.every(p => /^(0|[1-9][0-9]{0,2})$/.test(p) && Number(p) <= 255);
}
function renderDevices() {
  const list = $("devices"); list.replaceChildren();
  devices.forEach(address => {
    const li = document.createElement("li");
    const select = document.createElement("button"); select.type = "button"; select.textContent = address;
    select.addEventListener("click", () => { $("address").value = address; });
    const remove = document.createElement("button"); remove.type = "button"; remove.textContent = "删除";
    remove.setAttribute("aria-label", "删除 " + address);
    remove.addEventListener("click", () => { $("notice").textContent = ""; saveDevices(devices.filter(v => v !== address)); });
    li.append(select, " ", remove); list.append(li);
  });
}
try {
  const stored = JSON.parse(localStorage.getItem(key) || "[]");
  if (Array.isArray(stored)) devices = [...new Set(stored.filter(v => typeof v === "string" && validAddress(v)))];
} catch (_) { $("notice").textContent = "无法读取设备清单。"; }
renderDevices();
$("add").addEventListener("click", () => {
  const address = $("address").value.trim(); $("notice").textContent = "";
  if (!validAddress(address)) { $("notice").textContent = "请输入有效的 IPv4 地址。"; return; }
  if (devices.includes(address)) { $("notice").textContent = "该地址已存在。"; return; }
  if (saveDevices([...devices, address])) $("notice").textContent = "设备已添加。";
});
$("export").addEventListener("click", () => {
  const blob = new Blob([JSON.stringify({version: 1, devices}, null, 2)], {type: "application/json"});
  const url = URL.createObjectURL(blob), link = document.createElement("a");
  link.href = url; link.download = "lanpulse-devices.json"; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
});
$("import").addEventListener("change", async event => {
  const file = event.target.files[0]; if (!file) return;
  try {
    if (file.size > 256 * 1024) throw new Error("文件不能超过 256 KB");
    const data = JSON.parse(await file.text());
    if (!data || data.version !== 1 || !Array.isArray(data.devices) || data.devices.length > 4096 ||
        !data.devices.every(v => typeof v === "string" && validAddress(v))) throw new Error("文件格式或地址无效");
    const merged = [...new Set([...devices, ...data.devices])];
    if (saveDevices(merged)) $("notice").textContent = `导入完成，共 ${merged.length} 个地址。`;
  } catch (err) { $("notice").textContent = "导入失败：" + err.message; }
  finally { event.target.value = ""; }
});
for (const id of ["address", "timeout", "attempts"]) {
  const el = $(id);
  try { const value = localStorage.getItem("lanpulse." + id); if (value !== null) el.value = value;
    el.addEventListener("change", () => { try { localStorage.setItem("lanpulse." + id, el.value); } catch (_) {} });
  } catch (_) {}
}
function record(text) {
  const item = document.createElement("li"); item.textContent = new Date().toLocaleTimeString() + "　" + text;
  const history = $("history"); history.prepend(item); while (history.children.length > 20) history.lastElementChild.remove();
}
$("batch").addEventListener("click", async () => {
  if (!devices.length) { $("batch-status").textContent = "设备清单为空。"; return; }
  if (!$("timeout").reportValidity() || !$("attempts").reportValidity()) return;
  const queue = [...devices], timeout = Number($("timeout").value), attempts = Number($("attempts").value);
  $("batch").disabled = true; $("check").disabled = true;
  try {
    let next = 0, completed = 0;
    const runWorker = async () => {
      while (next < queue.length) {
        const address = queue[next++];
        try { record(await window.go.main.App.Ping(address, timeout, attempts)); }
        catch (err) { record(address + " 检测失败：" + err); }
        $("batch-status").textContent = `已检测 ${++completed}/${queue.length}`;
      }
    };
    await Promise.all(Array.from({ length: Math.min(4, queue.length) }, runWorker));
    $("batch-status").textContent = `已完成 ${queue.length} 台设备的检测。`;
  } finally { $("batch").disabled = false; $("check").disabled = false; }
});
$("clear").addEventListener("click", () => $("history").replaceChildren());
$("form").addEventListener("submit", async event => {
  event.preventDefault(); const button = $("check"), result = $("result"); button.disabled = true; $("batch").disabled = true; result.textContent = "正在检测…";
  try {
    result.textContent = await window.go.main.App.Ping($("address").value, Number($("timeout").value), Number($("attempts").value));
    record(result.textContent);
  } catch (err) { result.textContent = "检测失败：" + err; } finally { button.disabled = false; $("batch").disabled = false; }
});
