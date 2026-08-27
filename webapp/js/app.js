// Telegram Web App init
const tg = window.Telegram?.WebApp;
if (tg) {
    tg.ready();
    tg.expand();
}

// Tab switching
document.querySelectorAll('.nav-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        btn.classList.add('active');
        document.getElementById(`tab-${btn.dataset.tab}`).classList.add('active');
        loadTab(btn.dataset.tab);
    });
});

async function loadTab(tab) {
    switch (tab) {
        case 'profile': await loadProfile(); break;
        case 'map': await loadCities(); break;
        case 'work': await loadWorks(); break;
        case 'market': await loadMarket(); break;
    }
}

// Haptic feedback helper
function haptic(type) {
    if (tg?.HapticFeedback) {
        tg.HapticFeedback[type === 'error' ? 'notificationOccurred' : 'impactOccurred'](
            type === 'error' ? 'error' : 'medium'
        );
    }
}

// Profile
async function loadProfile() {
    const el = document.getElementById('profile-info');
    try {
        const data = await api.getProfile();
        const u = data.user;
        el.innerHTML = `
            <div class="stat-row"><span class="stat-label">Баланс</span><span class="stat-value">💰 ${u.balance.toLocaleString()} ₽</span></div>
            <div class="stat-row"><span class="stat-label">Уровень</span><span class="stat-value">📊 ${u.global_level}</span></div>
            <div class="stat-row"><span class="stat-label">XP</span><span class="stat-value">🎯 ${u.global_xp}</span></div>
            <div class="stat-row"><span class="stat-label">Серия</span><span class="stat-value">🔥 ${u.daily_streak} дн.</span></div>
            ${u.city_id ? `<div class="stat-row"><span class="stat-label">Город</span><span class="stat-value">🏙 #${u.city_id}</span></div>` : ''}
            ${u.corporation_id ? `<div class="stat-row"><span class="stat-label">Корпорация</span><span class="stat-value">🏢 #${u.corporation_id}</span></div>` : ''}
            ${u.active_job ? `<div class="stat-row"><span class="stat-label">Работа</span><span class="stat-value badge badge-yellow">🔨 ${u.active_job}</span></div>` : `<div class="stat-row"><span class="stat-label">Работа</span><span class="stat-value badge badge-green">Свободен</span></div>`}
        `;
    } catch (e) {
        el.innerHTML = '<p class="loading">Откройте /start в боте</p>';
    }
}

// Skills
async function loadSkills() {
    const el = document.getElementById('skills-list');
    try {
        const data = await api.getSkills();
        const skills = data.skills || [];
        el.innerHTML = skills.filter(s => s.xp > 0).map(s => `
            <div class="skill-item"><span>${s.direction}</span><span class="skill-xp">${s.xp} XP</span></div>
        `).join('') || '<p class="empty">Пока нет навыков. Начните работать!</p>';
    } catch (e) {
        el.innerHTML = '<p class="loading">Ошибка</p>';
    }
}

// Cities
async function loadCities() {
    const el = document.getElementById('cities-list');
    try {
        const data = await api.getCities();
        const cities = data.cities || [];
        el.innerHTML = cities.map(c => `
            <div class="city-item">
                <div class="name">${c.name}</div>
                <div class="info">Ур. ${c.level} | NPC: ${c.npc_population.toLocaleString()} | 👥 ${c.real_players}</div>
            </div>
        `).join('') || '<p class="empty">Публичных городов пока нет</p>';
    } catch (e) {
        el.innerHTML = '<p class="loading">Ошибка</p>';
    }
}

// Works
async function loadWorks() {
    const el = document.getElementById('work-list');
    try {
        const dirs = ['добыча', 'лес', 'топливо', 'энергетика', 'металлургия', 'строительство', 'химия', 'IT', 'торговля', 'агро'];
        let html = '';
        for (const dir of dirs) {
            const data = await api.getWorks(dir);
            const works = data.works || [];
            if (works.length === 0) continue;
            const w = works[0]; // Show first (easiest) work per direction
            html += `
                <div class="work-item">
                    <div class="name">${w.name}</div>
                    <div class="details">📂 ${dir} | ⏱ ${w.duration_minutes} мин | 💰 ${w.payout} ₽ | +${w.xp_reward} XP | +${w.resource_amount} ед.</div>
                    <button class="btn btn-primary" onclick="startWork('${w.id}')">Начать</button>
                </div>
            `;
        }
        el.innerHTML = html || '<p class="empty">Нет доступных работ</p>';
    } catch (e) {
        el.innerHTML = '<p class="loading">Ошибка</p>';
    }
}

// Market
async function loadMarket() {
    const el = document.getElementById('market-list');
    try {
        const profile = await api.getProfile();
        if (!profile.user.city_id) {
            el.innerHTML = '<p class="empty">Вступите в город, чтобы видеть рынок</p>';
            return;
        }
        const data = await api.getMarket(profile.user.city_id);
        const resources = data.resources || {};
        const names = { R1: 'Продовольствие', R2: 'Руда', R3: 'Древесина', R4: 'Топливо', R5: 'Энергия', R6: 'Металл', R7: 'Материалы', R8: 'Химикаты', R9: 'Технологии', R10: 'Потребтовары' };
        el.innerHTML = Object.entries(resources).map(([id, qty]) => `
            <div class="resource-item">
                <span class="resource-name">${names[id] || id}</span>
                <span class="resource-price">${qty.toLocaleString()} ед.</span>
            </div>
        `).join('') || '<p class="empty">Рынок пуст</p>';
    } catch (e) {
        el.innerHTML = '<p class="loading">Ошибка</p>';
    }
}

// Start work
async function startWork(workId) {
    try {
        await api.startWork(workId);
        haptic('success');
        alert('✅ Работа начата!');
    } catch (e) {
        haptic('error');
        alert('❌ ' + e.message);
    }
}

// Initial load
loadProfile();
loadSkills();
