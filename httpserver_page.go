package main

// configPageHTML 是内置的配置页（纯原生 HTML/CSS/JS，内联、离线可用，浏览器按需加载，几乎不占资源）。
const configPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>UFS Node 配置</title>
<style>
  :root{
    --primary:#2563EB; --primary-d:#1D4ED8;
    --bg:#F5F7FA; --card:#FFFFFF; --text:#1F2937; --muted:#6B7280;
    --green:#16A34A; --red:#DC2626; --amber:#D97706;
  }
  *{box-sizing:border-box}
  body{margin:0;font-family:"PingFang SC",-apple-system,"Microsoft YaHei",sans-serif;
    background:linear-gradient(160deg,#EEF2FF 0%,var(--bg) 40%);color:var(--text);min-height:100vh;padding:32px 16px}
  .wrap{max-width:680px;margin:0 auto}
  header{background:linear-gradient(135deg,var(--primary),var(--primary-d));color:#fff;border-radius:18px;
    padding:22px 26px;box-shadow:0 12px 30px rgba(37,99,235,.25);display:flex;align-items:center;justify-content:space-between}
  header h1{margin:0;font-size:22px;font-weight:600;letter-spacing:.5px}
  header p{margin:4px 0 0;opacity:.85;font-size:13px}
  .badge{font-size:13px;font-weight:600;padding:7px 14px;border-radius:999px;background:rgba(255,255,255,.18);transition:.2s}
  .badge.ok{background:rgba(22,163,74,.9)} .badge.err{background:rgba(220,38,38,.9)}
  .card{background:var(--card);border-radius:16px;padding:20px 22px;margin-top:18px;box-shadow:0 6px 18px rgba(17,24,39,.06)}
  .card h2{margin:0 0 14px;font-size:15px;font-weight:600;color:var(--primary);display:flex;align-items:center;gap:8px}
  .card h2::before{content:"";width:4px;height:16px;background:var(--primary);border-radius:2px;display:inline-block}
  label{display:block;font-size:13px;color:var(--muted);margin:12px 0 6px}
  input[type=text],input[type=number],input[type=password]{width:100%;padding:10px 12px;border:1px solid #E5E7EB;border-radius:10px;
    font-size:14px;color:var(--text);transition:.18s;background:#FCFCFD}
  input:focus{outline:none;border-color:var(--primary);box-shadow:0 0 0 3px rgba(37,99,235,.15)}
  input:disabled{background:#F3F4F6;color:#9CA3AF}
  .row{display:flex;gap:12px}.row>div{flex:1}
  .checkbox{display:flex;align-items:center;gap:8px;margin-top:14px;font-size:14px;color:var(--text)}
  .checkbox input{width:16px;height:16px;accent-color:var(--primary)}
  .actions{display:flex;gap:12px;margin-top:22px;flex-wrap:wrap}
  button{cursor:pointer;border:none;border-radius:11px;padding:12px 20px;font-size:14px;font-weight:600;transition:.18s;font-family:inherit}
  .btn-primary{background:linear-gradient(135deg,var(--primary),var(--primary-d));color:#fff;box-shadow:0 8px 18px rgba(37,99,235,.28)}
  .btn-primary:hover{transform:translateY(-1px);box-shadow:0 12px 22px rgba(37,99,235,.34)}
  .btn-ghost{background:#EEF2FF;color:var(--primary)}
  .btn-ghost:hover{background:#E0E7FF}
  .toast{position:fixed;left:50%;bottom:28px;transform:translateX(-50%) translateY(20px);background:#111827;color:#fff;
    padding:12px 20px;border-radius:12px;font-size:14px;opacity:0;pointer-events:none;transition:.25s;max-width:90vw}
  .toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
  .hint{font-size:12px;color:var(--muted);margin-top:6px;line-height:1.5}
  footer{text-align:center;color:var(--muted);font-size:12px;margin-top:22px}
</style>
</head>
<body>
<div class="wrap">
  <header>
    <div>
      <h1>UFS Node</h1>
      <p>RemoteStorage 上报代理 · 本地配置</p>
    </div>
    <div id="badge" class="badge">加载中…</div>
  </header>

  <form id="form" autocomplete="off">
    <div class="card">
      <h2>本节点</h2>
      <label>UUID（节点唯一标识，自动生成）</label>
      <input type="text" id="uuid" disabled>
      <label>显示名称</label>
      <input type="text" id="name" placeholder="节点名称">
    </div>

    <div class="card">
      <h2>RemoteStorage 服务器</h2>
      <label>服务器地址 Server（存储根，含用户名路径）</label>
      <input type="text" id="rs_server" placeholder="https://storage.5apps.com/weijia">
      <div class="hint">例如 5apps 为 https://storage.5apps.com/&lt;用户名&gt;；最终 PUT 地址 = 该地址 + 路径模板。</div>
      <div class="row">
        <div>
          <label>用户 User</label>
          <input type="text" id="rs_user" placeholder="me">
        </div>
        <div>
          <label>Scope</label>
          <input type="text" id="rs_scope" placeholder="/ufs-nodes/">
        </div>
      </div>
      <label>Bearer Token</label>
      <input type="password" id="rs_token" placeholder="鉴权令牌">
      <label>路径模板 PathTemplate（支持 {user} {uuid}）</label>
      <input type="text" id="rs_path" placeholder="/ufs-nodes/{uuid}.json">
      <div class="hint">实际 PUT 地址为：<span id="resolved"></span></div>
    </div>

    <div class="card">
      <h2>上报设置</h2>
      <label>上报间隔（分钟）</label>
      <input type="number" id="report_interval" min="1" placeholder="15">
      <label class="checkbox"><input type="checkbox" id="report_extra"> 附带本机基础信息（hostname / 操作系统 / 版本）</label>
    </div>

    <div class="card">
      <h2>本地服务</h2>
      <label>HTTP 监听地址（仅本机）</label>
      <input type="text" id="http_listen" placeholder="127.0.0.1:9801">
      <div class="hint">修改监听地址保存后会自动重启本地服务。</div>
    </div>

    <div class="actions">
      <button type="submit" class="btn-primary">保存并应用</button>
      <button type="button" class="btn-ghost" id="updateBtn">立即上报</button>
    </div>
  </form>

  <footer>UFS Node · 状态每间隔自动上报至 RemoteStorage 服务器</footer>
</div>

<div class="toast" id="toast"></div>

<script>
const $ = id => document.getElementById(id);
function toast(msg){
  const t=$('toast'); t.textContent=msg; t.classList.add('show');
  clearTimeout(t._t); t._t=setTimeout(()=>t.classList.remove('show'),2600);
}
function fill(c){
  $('uuid').value=c.uuid||'';
  $('name').value=c.name||'';
  $('rs_server').value=c.remotestorage?.server||'';
  $('rs_user').value=c.remotestorage?.user||'';
  $('rs_scope').value=c.remotestorage?.scope||'';
  $('rs_token').value=c.remotestorage?.token||'';
  $('rs_path').value=c.remotestorage?.path_template||'';
  $('report_interval').value=c.report?.interval_minutes||'';
  $('report_extra').checked=!!c.report?.extra_info;
  $('http_listen').value=c.http?.listen||'';
  refreshResolved();
}
function collect(){
  return {
    uuid:$('uuid').value, name:$('name').value,
    remotestorage:{
      server:$('rs_server').value, user:$('rs_user').value,
      scope:$('rs_scope').value, token:$('rs_token').value,
      path_template:$('rs_path').value
    },
    report:{ interval_minutes:parseInt($('report_interval').value,10)||0, extra_info:$('report_extra').checked },
    http:{ listen:$('http_listen').value }
  };
}
function refreshResolved(){
  const s=$('rs_server').value.replace(/\/+$/,'');
  const p=($('rs_path').value||'/ufs-nodes/{uuid}.json').replace(/\{user\}/g,$('rs_user').value).replace(/\{uuid\}/g,$('uuid').value);
  $('resolved').textContent=s+p;
}
['rs_server','rs_user','rs_path','uuid'].forEach(id=>$(id).addEventListener('input',refreshResolved));

async function load(){
  try{
    const [cfg, st] = await Promise.all([
      fetch('/api/config').then(r=>r.json()),
      fetch('/api/status').then(r=>r.json())
    ]);
    fill(cfg);
    updateBadge(st.status);
  }catch(e){ toast('加载失败: '+e.message); }
}
function updateBadge(s){
  const b=$('badge');
  if(!s || !s.last_attempt){ b.textContent='未上报'; b.className='badge'; return; }
  const t=new Date(s.last_attempt).toLocaleString();
  if(s.last_success){ b.textContent='上次上报成功 '+t; b.className='badge ok'; }
  else { b.textContent='上次上报失败 '+t; b.className='badge err'; }
}

$('form').addEventListener('submit', async e=>{
  e.preventDefault();
  try{
    const r=await fetch('/api/config',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(collect())});
    const data=await r.json();
    if(!r.ok){ toast('保存失败: '+(data.error||r.status)); return; }
    toast('已保存并应用');
    load();
  }catch(e){ toast('保存失败: '+e.message); }
});
$('updateBtn').addEventListener('click', async ()=>{
  try{
    const r=await fetch('/api/update',{method:'POST'});
    const data=await r.json();
    if(data.ok){ toast('上报成功'); } else { toast('上报失败: '+(data.error||'')); }
    load();
  }catch(e){ toast('上报失败: '+e.message); }
});
load();
</script>
</body>
</html>`
