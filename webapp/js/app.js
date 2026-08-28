const tg=window.Telegram?.WebApp;
if(tg){tg.ready();tg.expand();}

const D=[
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

const RN={
R1:{n:'Продовольствие',e:'🍞',c:'#ff9800'},
R2:{n:'Руда',e:'⛏',c:'#795548'},
R3:{n:'Древесина',e:'🪵',c:'#4caf50'},
R4:{n:'Топливо',e:'⛽',c:'#f44336'},
R5:{n:'Энергия',e:'⚡',c:'#ffc107'},
R6:{n:'Металл',e:'🔩',c:'#607d8b'},
R7:{n:'Материалы',e:'🧱',c:'#ff5722'},
R8:{n:'Химикаты',e:'🧪',c:'#9c27b0'},
R9:{n:'Технологии',e:'💻',c:'#2196f3'},
R10:{n:'Потребтовары',e:'🛒',c:'#e91e63'}
};

let curDir=null;

// Skeleton helper
function sk(n){return Array(n).fill('<div class="skeleton skeleton-row"></div>').join('');}

// Navigation
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
function toast(msg,type){
    const t=document.createElement('div');
    t.className='toast toast-'+type;
    t.textContent=msg;
    document.body.appendChild(t);
    setTimeout(()=>t.remove(),type==='ok'?2000:3000);
}

// Profile
async function loadProfile(){
    const hero=document.getElementById('hero-profile');
    const stats=document.getElementById('profile-stats');
    stats.innerHTML=sk(4);
    try{
        const d=await api.getProfile();
        const u=d.user;
        const xp=u.global_xp||0;
        const lvl=u.global_level||1;
        const nextXp=lvl*100;
        const xpPct=Math.min(Math.round(xp/nextXp*100),100);

        document.getElementById('hero-name').textContent='Игрок #'+u.telegram_user_id;
        document.getElementById('hero-level').textContent='Уровень '+lvl+' · '+xp+' XP';
        document.getElementById('hero-xp-bar').innerHTML='<div class="hero-xp-fill" style="width:'+xpPct+'%"></div>';

        let activeHtml='';
        if(u.active_job){
            activeHtml='<div class="active-banner"><span class="active-banner-icon">🔨</span><div class="active-banner-info"><div class="active-banner-title">Работаю: '+u.active_job+'</div><div class="active-banner-timer">Выполняется...</div></div></div>';
        }

        let cityHtml=u.city_id?'<span>🏙 Город #'+u.city_id+'</span>':'<span style="opacity:.5">🏙 Не выбран</span>';
        let corpHtml=u.corporation_id?'<div class="stat-row"><span class="stat-label">Корпорация</span><span class="stat-value">🏢 #'+u.corporation_id+'</span></div>':'';

        stats.innerHTML=activeHtml+
            '<div class="stat-row"><span class="stat-label">Баланс</span><span class="stat-value">💰 '+(u.balance||0).toLocaleString()+' ₽</span></div>'+
            '<div class="stat-row"><span class="stat-label">Серия</span><span class="stat-value">🔥 '+u.daily_streak+' / 7 дн.</span></div>'+
            '<div class="stat-row"><span class="stat-label">Город</span><span class="stat-value">'+cityHtml+'</span></div>'+
            corpHtml+
            '<div class="stat-row"><span class="stat-label">Работа</span><span class="stat-value">'+(u.active_job?'<span class="badge badge-yellow">В работе</span>':'<span class="badge badge-green">Свободен</span>')+'</span></div>';
    }catch(e){
        document.getElementById('hero-name').textContent='Откройте /start';
        document.getElementById('hero-level').textContent='';
        document.getElementById('hero-xp-bar').innerHTML='';
        stats.innerHTML='<p class="empty">⚠️ '+e.message+'</p>';
    }
}

// Skills
async function loadSkills(){
    const el=document.getElementById('skills-list');
    el.innerHTML=sk(5);
    try{
        const d=await api.getSkills();
        const skMap={};
        (d.skills||[]).forEach(s=>{skMap[s.direction]=s.xp;});
        el.innerHTML=D.map(d=>{
            const xp=skMap[d.id]||0;
            const nl=Math.ceil(xp/100)*100||100;
            return '<div class="skill-item"><span class="skill-name">'+d.e+' '+d.n+'</span><div class="skill-bar-wrap">'+pb(xp,nl)+'</div><span class="skill-xp">'+xp+'</span></div>';
        }).join('');
    }catch(e){el.innerHTML='<p class="empty">Загрузка...</p>';}
}

// Cities
async function loadCities(){
    const el=document.getElementById('cities-list');
    el.innerHTML=sk(3);
    try{
        const d=await api.getCities();
        const cs=d.cities||[];
        el.innerHTML=cs.map(c=>'<div class="city-card"><div class="city-header"><span class="city-name">🏙 '+c.name+'</span><span class="city-level badge badge-green">'+c.level+'</span></div><div class="city-stats"><span>👥 '+c.real_players+'</span><span>🤖 '+(c.npc_population||0).toLocaleString()+'</span></div></div>').join('')||'<p class="empty">Публичных городов пока нет</p>';
    }catch(e){el.innerHTML='<p class="empty">Ошибка загрузки</p>';}
}

// Work Directions
async function loadWorkDirs(){
    const el=document.getElementById('work-list');
    if(curDir){await loadWorks(curDir);return;}
    el.innerHTML='<div style="padding:12px">'+sk(8)+'</div>';
    try{
        let skMap={};
        try{const s=await api.getSkills();(s.skills||[]).forEach(x=>{skMap[x.direction]=x.xp;});}catch(_){}
        el.innerHTML=D.map(d=>{
            const xp=skMap[d.id]||0;
            const nl=Math.ceil(xp/100)*100||100;
            return '<div class="dir-card" onclick="openDir(\''+d.id+'\')"><span class="dir-emoji">'+d.e+'</span><div class="dir-info"><span class="dir-name">'+d.n+'</span><div class="dir-xp">'+pb(xp,nl)+' <span style="margin-left:4px">'+xp+' XP</span></div></div><span class="dir-arrow">›</span></div>';
        }).join('');
    }catch(e){el.innerHTML='<p class="empty">Ошибка</p>';}
}

function openDir(id){curDir=id;haptic('tap');loadWorks(id);}

async function loadWorks(dirId){
    const el=document.getElementById('work-list');
    const dir=D.find(d=>d.id===dirId);
    el.innerHTML='<div style="padding:12px">'+sk(6)+'</div>';
    try{
        const d=await api.getWorks(dirId);
        const ws=d.works||[];
        let skMap={};
        try{const s=await api.getSkills();(s.skills||[]).forEach(x=>{skMap[x.direction]=x.xp;});}catch(_){}
        const uxp=skMap[dirId]||0;
        let h='<div class="dir-header" onclick="curDir=null;loadWorkDirs();"><span class="dir-back">‹</span> '+(dir?dir.e+' '+dir.n:dirId)+'</div>';
        ws.forEach(w=>{
            const ok=uxp>=(w.required_xp||0);
            const lock=!ok&&(w.required_xp||0)>0;
            h+='<div class="work-card'+(lock?' locked':'')+'"><div class="work-header"><span class="work-name">'+(lock?'🔒':'✅')+' '+w.name+'</span></div><div class="work-details"><span>⏱ '+w.duration_minutes+' мин</span><span>💰 '+w.payout+' ₽</span><span>🎯 +'+w.xp_reward+' XP</span></div><div class="work-details"><span>📦 +'+w.resource_amount+' ед.</span>'+(lock?'<span class="badge badge-red">'+w.required_xp+' XP</span>':'')+'</div>'+(ok?'<button class="btn btn-primary btn-full work-start-btn" onclick="startWork(\''+w.id+'\')">▶ Начать</button>':'')+'</div>';
        });
        el.innerHTML=h;
    }catch(e){el.innerHTML='<p class="empty">'+e.message+'</p><div class="dir-header" onclick="curDir=null;loadWorkDirs();"><span class="dir-back">‹</span> Назад</div>';}
}

// Market
async function loadMarket(){
    const el=document.getElementById('market-list');
    el.innerHTML=sk(5);
    try{
        let cid=null;
        try{const p=await api.getProfile();cid=p.user?.city_id;}catch(_){}
        if(!cid){el.innerHTML='<p class="empty">🏙 Вступите в город, чтобы видеть рынок</p>';return;}
        const d=await api.getMarket(cid);
        const rs=d.resources||[];
        el.innerHTML=rs.map(r=>{
            const info=RN[r.resource_id]||{n:r.resource_id,e:'📦',c:'#666'};
            return '<div class="resource-item"><div class="resource-left"><div class="resource-icon" style="background:'+info.c+'20;color:'+info.c+'">'+info.e+'</div><span class="resource-name">'+info.n+'</span></div><span class="resource-price">'+(r.stock||0).toLocaleString()+' ед.</span></div>';
        }).join('')||'<p class="empty">Рынок пуст</p>';
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

// Init
loadProfile();loadSkills();
