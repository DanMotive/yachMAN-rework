const API_BASE = window.location.origin;

const api = {
    async get(path) {
        const resp = await fetch(`${API_BASE}${path}`);
        if (!resp.ok) throw new Error(`API ${resp.status}`);
        return resp.json();
    },

    async post(path, body) {
        const resp = await fetch(`${API_BASE}${path}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        if (!resp.ok) {
            const err = await resp.json().catch(() => ({}));
            throw new Error(err.error || `API ${resp.status}`);
        }
        return resp.json();
    },

    getProfile: () => api.get('/api/me'),
    getSkills: () => api.get('/api/me/skills'),
    getCities: () => api.get('/api/cities'),
    getCity: (id) => api.get(`/api/cities/${id}`),
    getWorks: (dir) => api.get(`/api/works/${dir || ''}`),
    getMarket: (cityId) => api.get(`/api/resources/${cityId}`),
    getEvents: () => api.get('/api/events'),
    startWork: (workId) => api.post('/api/work/start', { work_id: workId }),
    getEducation: () => api.get('/api/me/education'),
    enroll: (programId) => api.post('/api/study', { program_id: programId }),
    study: (programId) => api.post('/api/study/lesson', { program_id: programId }),
};
