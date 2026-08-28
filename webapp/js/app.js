const tg = window.Telegram?.WebApp;
if (tg) { tg.ready(); tg.expand(); }

const D = [
    {id:'добыча',e:'⛏',n:'Добыча'},{id:'лес',e:'🌲',n:'Лес'},
    {id:'топливо',e:'⛽',n:'Топливо'},{id:'энергетика',e:'⚡',n:'Энергетика'},
    {id:'металлургия',e:'🔥',n:'Металлургия'},{id:'строительство',e:'🏗',n:'Строительство'},
    {id:'химия',e:'🧪',n:'Химия'},{id:'IT',e:'💻',n:'IT'},
    {id:'торговля',e:'🏪',n:'Торговля'},{id:'агро',e:'🌾',n:'Агро'},
    {id:'транспорт',e:'🚛',n:'Транспорт'},{id:'питание',e:'🍽',n:'Питание'},
    {id:'ремонт',e:'🔧',n:'Ремонт'},{id:'медицина',e:'🏥',n:'Медицина'},
    {id:'образование',e:'📚',n:'Образование'},{id:'наука',e:'🔬',n:'Наука'},
    {id:'безопасность',e:'🛡',n:'Безопасность'},{id:'медиа',e:'📺',n:'Медиа'},
    {id:'коммунальные услуги',e:'🏢',n:'Коммунальные услуги'},
    {id:'переработка',e:'♻',n:'Переработка'}
];
const RN={R1:'Продовольствие',R2:'Руда',R3:'Древесина',R4:'Топливо',R5:'Энергия',R6:'Металл',R7:'Материалы',R8:'Химикаты',R9:'Технологии',R10:'Потребтовары'};

let curDir=null;

document.querySelectorAll('.nav-btn').forEach(b=>{b.addEventListener('click',()=>{
    document.querySelectorAll('.nav-btn').forEach(x=>x.classList.remove('active'));
    document.querySelectorAll('.tab').forEach(x=>x.classList.remove('active'));
    b.classList.add('active');
    document.getElementById('tab-'+b.dataset.tab).classList.add('active');
    loadTab(b.dataset.tab);
});});

async function loadTab(t){
    if(t==='profile'){await loadProfile();loadSkills();}
    else if(t==='map')await loadCities();
    else if(t==='work')await loadWorkDirs();
    else if(t==='market')await loadMarket();
}

function haptic(t){if(tg?.HapticFeedback){t==='error'?tg.HapticFeedback.notificationOccurred('error'):tg.HapticFeedback.impactOccurred('medium');}}
function pb(c,m){if(!m)m=1;const pct=Math.min(Math.round(c/m*100),100);return '<div class="progress-bar"><div class="progress-fill" style="width:'+pct+'%"></div></div>';}

// Profile
async function loadProfile(){
    const el=document.getElementById('profile-info');
    try{
        const d=await api.getProfile();
        const u=d.user;
        el.innerHTML=`
            <div class="stat-row"><span class="stat-label">Баланс</span><span class="stat-value">💰 ${(u.balance||0).toLocaleString()} ₽</span></div>
            <div class="stat-row"><span class="stat-label">Уровень</span><span class="stat-value">📊 ${u.global_level||1}</span></div>
            <div class="stat-row"><span class="stat-label">XP</span><span class="stat-value">🎯 ${(u.global_xp||0).toLocaleString()}</span></div>
            <div class="stat-row"><span class="stat-label">Серия</span><span class="stat-value">🔥 ${u.daily_streak||0} / 7</span></div>
            ${u.city_id?`<div class="stat-row"><span class="stat-label">Город</span><span class="stat-value">🏙 #${u.city_id}</span></div>`:'<div class="stat-row"><span class="stat-label">Город</span><span class="stat-value" style="opacity:0.5">не выбран</span></div>'}
            ${u.corporation_id?`<div class="stat-row"><span class="stat-label">Корпорация</span><span class="stat-value">🏢 #${u.corporation_id}</span></div>`:''}
            <div class="stat-row"><span class="stat-label">Работа</span><span class="stat-value">${u.active_job?`<span class="badge badge-yellow">🔨 ${u.active_job}</span>`:'<span class="badge badge-green">Свободен</span>'}</span></div>`;
    }catch(e){
        el.innerHTML='<p class="loading">⚠️ Откройте /start в боте</p>';
    }
}

// Skills
async function loadSkills(){
    const el=document.getElementById('skills-list');
    try{
        const d=await api.getSkills();
        const sk=d.skills||[];
        el.innerHTML=sk.filter(s=>s.xp>0).map(s=>{
            const nl=Math.ceil(s.xp/100)*100;
            return `<div class="skill-item"><span>${s.direction}</span><span class="skill-xp">${pb(s.xp,nl,8)} ${s.xp}</span></div>`;
        }).join('')||'<p class="empty">Начните работать!</p>';
    }catch(e){el.innerHTML='';}
}

// Cities
async function loadCities(){
    const el=document.getElementById('cities-list');
    try{
        const d=await api.getCities();
        const cs=d.cities||[];
        el.innerHTML=cs.map(c=>`
            <div class="city-card">
                <div class="city-header"><span class="city-name">🏙 ${c.name}</span><span class="city-level badge badge-green">${c.level}</span></div>
                <div class="city-stats"><span>👥 ${c.real_players}</span><span>🤖 ${(c.npc_population||0).toLocaleString()}</span></div>
            </div>`).join('')||'<p class="empty">Публичных городов пока нет</p>';
    }catch(e){el.innerHTML='<p class="empty">Ошибка загрузки</p>';}
}

// Work Directions
async function loadWorkDirs(){
    const el=document.getElementById('work-list');
    if(curDir){await loadWorks(curDir);return;}
    try{
        let sk={};
        try{const s=await api.getSkills();(s.skills||[]).forEach(x=>{sk[x.direction]=x.xp;});}catch(_){}
        el.innerHTML=D.map(d=>{
            const xp=sk[d.id]||0;
            const nl=Math.ceil(xp/100)*100||100;
            return `<div class="dir-card" onclick="openDir('${d.id}')">
                <span class="dir-emoji">${d.e}</span>
                <div class="dir-info"><span class="dir-name">${d.n}</span><span class="dir-xp">${pb(xp,nl,6)} ${xp} XP</span></div>
                <span class="dir-arrow">›</span></div>`;
        }).join('');
    }catch(e){el.innerHTML='<p class="empty">Ошибка</p>';}
}

function openDir(id){curDir=id;haptic('tap');loadWorks(id);}

async function loadWorks(dirId){
    const el=document.getElementById('work-list');
    const dir=D.find(d=>d.id===dirId);
    try{
        const d=await api.getWorks(dirId);
        const ws=d.works||[];
        let sk={};
        try{const s=await api.getSkills();(s.skills||[]).forEach(x=>{sk[x.direction]=x.xp;});}catch(_){}
        const uxp=sk[dirId]||0;
        let h=`<div class="dir-header" onclick="curDir=null;loadWorkDirs();"><span class="dir-back">‹</span> ${dir?dir.e+' '+dir.n:dirId}</div>`;
        ws.forEach(w=>{
            const ok=uxp>=(w.required_xp||0);
            const lock=!ok&&(w.required_xp||0)>0;
            h+=`<div class="work-card${lock?' locked':''}">
                <div class="work-header"><span class="work-name">${lock?'🔒':'✅'} ${w.name}</span></div>
                <div class="work-details"><span>⏱ ${w.duration_minutes}м</span><span>💰 ${w.payout}₽</span><span>🎯+${w.xp_reward}</span></div>
                <div class="work-details"><span>📦+${w.resource_amount}</span>${lock?`<span class="badge badge-red">${w.required_xp}XP</span>`:''}</div>
                ${ok?`<button class="btn btn-primary work-start-btn" onclick="startWork('${w.id}')">▶ Начать</button>`:''}</div>`;
        });
        el.innerHTML=h;
    }catch(e){el.innerHTML=`<p class="empty">${e.message}</p><div class="dir-header" onclick="curDir=null;loadWorkDirs();"><span class="dir-back">‹</span> Назад</div>`;}
}

// Market
async function loadMarket(){
    const el=document.getElementById('market-list');
    try{
        let cid=null;
        try{const p=await api.getProfile();cid=p.user?.city_id;}catch(_){}
        if(!cid){el.innerHTML='<p class="empty">🏙 Вступите в город</p>';return;}
        const d=await api.getMarket(cid);
        const rs=d.resources||[];
        el.innerHTML=rs.map(r=>`<div class="resource-item"><span class="resource-name">${RN[r.resource_id]||r.resource_id}</span><span class="resource-price">${(r.stock||0).toLocaleString()} ед.</span></div>`).join('')||'<p class="empty">Рынок пуст</p>';
    }catch(e){el.innerHTML='<p class="empty">'+e.message+'</p>';}
}

// Start Work
async function startWork(wid){
    haptic('success');
    try{
        await api.startWork(wid);
        toast('✅ Работа начата!','ok');
        curDir=null;loadWorkDirs();
    }catch(e){haptic('error');toast('❌ '+e.message,'err');}
}

function toast(msg,type){
    const t=document.createElement('div');
    t.className='toast toast-'+type;
    t.textContent=msg;
    document.body.appendChild(t);
    setTimeout(()=>t.remove(),type==='ok'?2000:3000);
}

// Init
loadProfile();loadSkills();
