// API client for Vigil backend

const BASE = '/api';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
    const res = await fetch(`${BASE}${path}`, {
        headers: { 'Content-Type': 'application/json' },
        ...options,
    });
    if (!res.ok) {
        const body = await res.text();
        throw new Error(`API ${res.status}: ${body}`);
    }
    if (res.status === 204) return undefined as T;
    return res.json();
}

// --- Types ------------------------------------------------------------------

export interface Status {
    status: string;
    uptime: string;
}

export interface Schedule {
    id: number;
    cron_expr: string;
    camera_id: string;
    hook_path: string;
    enabled: boolean;
}

export interface CaptureLog {
    id: number;
    schedule_id: number;
    timestamp: string;
    filepath: string;
    hook_status: string;
    hook_output: string;
}

// --- Endpoints --------------------------------------------------------------

export const api = {
    getStatus: () => request<Status>('/status'),

    getSchedules: () => request<Schedule[]>('/schedules'),
    createSchedule: (s: Omit<Schedule, 'id'>) =>
        request<Schedule>('/schedules', { method: 'POST', body: JSON.stringify(s) }),
    updateSchedule: (id: number, s: Partial<Schedule>) =>
        request<Schedule>(`/schedules/${id}`, { method: 'PUT', body: JSON.stringify(s) }),
    deleteSchedule: (id: number) =>
        request<void>(`/schedules/${id}`, { method: 'DELETE' }),

    getConfig: () => request<Record<string, string>>('/config'),
    setConfig: (cfg: Record<string, string>) =>
        request<Record<string, string>>('/config', { method: 'PUT', body: JSON.stringify(cfg) }),

    getCaptures: (date?: string, limit?: number) => {
        const params = new URLSearchParams();
        if (date) params.set('date', date);
        if (limit) params.set('limit', String(limit));
        const qs = params.toString();
        return request<CaptureLog[]>(`/captures${qs ? `?${qs}` : ''}`);
    },

    triggerCapture: () =>
        request<{ status: string; camera_id: string }>('/captures/trigger', { method: 'POST' }),

    deleteCaptures: (ids: number[]) =>
        request<{ deleted: number }>('/captures', {
            method: 'DELETE',
            body: JSON.stringify({ ids }),
        }),

    getCaptureUrl: (filepath: string) => {
        // filepath is like /data/captures/2024-10-27/09-00-00.jpg
        // We need to extract date and filename
        const parts = filepath.replace(/\\/g, '/').split('/');
        const file = parts[parts.length - 1];
        const date = parts[parts.length - 2];
        return `${BASE}/captures/${date}/${file}`;
    },
};
