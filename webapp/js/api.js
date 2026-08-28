const API_BASE = window.location.origin;

// Get Telegram Web App initData for auth
function getInitData() {
    const tg = window.Telegram?.WebApp;
    return tg?.initData || '';
}

const api = {
    async get(path, requiresAuth = false) {
        const headers = {};
        if (requiresAuth) {
            const initData = getInitData();
            if (initData) headers['X-Telegram-Init-Data'] = initData;
        }
        const resp = await fetch(`${API_BASE}${path}`, { headers });
        if (!resp.ok) {
            const err = await resp.json().catch(() => ({}));
            throw new Error(err.error || `HTTP ${resp.status}`);
        }
        return resp.json();
    },

    async post(path, body, requiresAuth = false) {
        const headers = { 'Content-Type': 'application/json' };
        if (requiresAuth) {
            const initData = getInitData();
            if (initData) headers['X-Telegram-Init-Data'] = initData;
        }
        const resp = await fetch(`${API_BASE}${path}`, {
            method: 'POST',
            headers,
            body: JSON.stringify(body)
        });
        if (!resp.ok) {
            const err = await resp.json().catch(() => ({}));
            throw new Error(err.error || `HTTP ${resp.status}`);
        }
        return resp.json();
    },

    // Public endpoints
    getCities: () => api.get('/api/cities'),
    getCity: (id) => api.get(`/api/cities/${id}`),
    getWorks: (dir) => api.get(`/api/works/${dir || ''}`),
    getMarket: (cityId) => api.get(`/api/resources/${cityId}`),
    getEvents: () => api.get('/api/events'),
    getDirections: () => api.get('/api/works'),

    // Protected endpoints (need Telegram init data)
    getProfile: () => api.get('/api/me', true),
    getSkills: () => api.get('/api/me/skills', true),
    startWork: (workId) => api.post('/api/work/start', { work_id: workId }, true),
};
