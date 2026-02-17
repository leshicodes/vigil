import { useEffect, useState } from 'react';
import { Plus, Pencil, Trash2, Clock } from 'lucide-react';
import { api, type Schedule } from '../lib/api';
import Modal from '../components/Modal';
import ScheduleForm from '../components/ScheduleForm';
import { toast } from '../components/Toast';

export default function Config() {
    const [schedules, setSchedules] = useState<Schedule[]>([]);
    const [config, setConfig] = useState<Record<string, string>>({});
    const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
    const [showForm, setShowForm] = useState(false);

    const fetchData = async () => {
        try {
            const [scheds, cfg] = await Promise.all([
                api.getSchedules(),
                api.getConfig(),
            ]);
            setSchedules(scheds);
            setConfig(cfg);
        } catch (err) {
            console.error('Config fetch error:', err);
        }
    };

    useEffect(() => { fetchData(); }, []);

    const handleCreate = async (data: Omit<Schedule, 'id'>) => {
        try {
            await api.createSchedule(data);
            toast('success', 'Schedule created');
            setShowForm(false);
            fetchData();
        } catch (err) {
            toast('error', `Failed to create: ${err}`);
        }
    };

    const handleUpdate = async (data: Omit<Schedule, 'id'>) => {
        if (!editingSchedule) return;
        try {
            await api.updateSchedule(editingSchedule.id, { ...data, id: editingSchedule.id } as Schedule);
            toast('success', 'Schedule updated');
            setEditingSchedule(null);
            fetchData();
        } catch (err) {
            toast('error', `Failed to update: ${err}`);
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await api.deleteSchedule(id);
            toast('success', 'Schedule deleted');
            fetchData();
        } catch (err) {
            toast('error', `Failed to delete: ${err}`);
        }
    };

    const handleConfigSave = async (key: string, value: string) => {
        try {
            await api.setConfig({ [key]: value });
            toast('success', `Config "${key}" updated`);
            setConfig((prev) => ({ ...prev, [key]: value }));
        } catch (err) {
            toast('error', `Failed to save config: ${err}`);
        }
    };

    return (
        <div className="space-y-8 animate-fade-in">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-bold text-text-primary">Configuration</h1>
                <p className="text-sm text-text-muted mt-1">Manage schedules and system settings</p>
            </div>

            {/* Schedules Section */}
            <section>
                <div className="flex items-center justify-between mb-4">
                    <h2 className="text-lg font-semibold text-text-primary">Schedules</h2>
                    <button
                        onClick={() => setShowForm(true)}
                        className="flex items-center gap-2 px-3 py-2 rounded-lg bg-gradient-to-r from-accent-cyan to-accent-teal text-surface-base text-sm font-semibold hover:opacity-90 transition-opacity"
                    >
                        <Plus className="w-4 h-4" />
                        Add Schedule
                    </button>
                </div>

                {schedules.length === 0 ? (
                    <div className="glass rounded-xl p-8 text-center">
                        <Clock className="w-10 h-10 text-text-muted mx-auto mb-3" />
                        <p className="text-text-secondary font-medium">No schedules</p>
                        <p className="text-sm text-text-muted mt-1">Create a schedule to start capturing</p>
                    </div>
                ) : (
                    <div className="space-y-3">
                        {schedules.map((sched) => (
                            <div
                                key={sched.id}
                                className="glass rounded-xl px-5 py-4 flex items-center gap-4 hover:border-border-bright transition-colors"
                            >
                                {/* Status indicator */}
                                <div className={`w-2.5 h-2.5 rounded-full shrink-0 ${sched.enabled ? 'bg-status-ok' : 'bg-text-muted'}`} />

                                {/* Info */}
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-mono font-medium text-text-primary">{sched.cron_expr}</p>
                                    <p className="text-xs text-text-muted mt-0.5">
                                        Camera: <span className="text-text-secondary">{sched.camera_id}</span>
                                        {sched.hook_path && (
                                            <> · Hook: <span className="text-text-secondary">{sched.hook_path}</span></>
                                        )}
                                    </p>
                                </div>

                                {/* Actions */}
                                <div className="flex items-center gap-1.5">
                                    <button
                                        onClick={() => setEditingSchedule(sched)}
                                        className="p-2 rounded-lg text-text-muted hover:text-accent-cyan hover:bg-accent-cyan/10 transition-colors"
                                        title="Edit"
                                    >
                                        <Pencil className="w-4 h-4" />
                                    </button>
                                    <button
                                        onClick={() => handleDelete(sched.id)}
                                        className="p-2 rounded-lg text-text-muted hover:text-status-error hover:bg-status-error/10 transition-colors"
                                        title="Delete"
                                    >
                                        <Trash2 className="w-4 h-4" />
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </section>

            {/* Config Section */}
            <section>
                <h2 className="text-lg font-semibold text-text-primary mb-4">System Settings</h2>
                <div className="glass rounded-xl divide-y divide-border-subtle">
                    <ConfigRow
                        label="Camera Driver"
                        description="Default camera driver for new schedules"
                        value={config['camera_driver'] ?? 'mock'}
                        type="select"
                        options={[
                            { value: 'mock', label: 'Mock (Test)' },
                            { value: 'libcamera', label: 'libcamera (RPi CSI)' },
                            { value: 'fswebcam', label: 'fswebcam (USB)' },
                            { value: 'ffmpeg', label: 'ffmpeg (USB/DirectShow)' },
                        ]}
                        onSave={(v) => handleConfigSave('camera_driver', v)}
                    />
                    <ConfigRow
                        label="Hook Timeout"
                        description="Maximum time (seconds) for hook scripts to complete"
                        value={config['hook_timeout'] ?? '60'}
                        type="number"
                        onSave={(v) => handleConfigSave('hook_timeout', v)}
                    />
                </div>
            </section>

            {/* Create Modal */}
            <Modal open={showForm} onClose={() => setShowForm(false)} title="New Schedule">
                <ScheduleForm onSubmit={handleCreate} onCancel={() => setShowForm(false)} />
            </Modal>

            {/* Edit Modal */}
            <Modal open={!!editingSchedule} onClose={() => setEditingSchedule(null)} title="Edit Schedule">
                {editingSchedule && (
                    <ScheduleForm
                        initial={editingSchedule}
                        onSubmit={handleUpdate}
                        onCancel={() => setEditingSchedule(null)}
                    />
                )}
            </Modal>
        </div>
    );
}

// --- Helper ---

function ConfigRow({ label, description, value, type, options, onSave }: {
    label: string;
    description: string;
    value: string;
    type: 'text' | 'number' | 'select';
    options?: { value: string; label: string }[];
    onSave: (val: string) => void;
}) {
    const [val, setVal] = useState(value);
    const changed = val !== value;

    useEffect(() => { setVal(value); }, [value]);

    return (
        <div className="px-5 py-4 flex items-center gap-4 flex-wrap">
            <div className="flex-1 min-w-[200px]">
                <p className="text-sm font-medium text-text-primary">{label}</p>
                <p className="text-xs text-text-muted mt-0.5">{description}</p>
            </div>
            <div className="flex items-center gap-2">
                {type === 'select' && options ? (
                    <select
                        value={val}
                        onChange={(e) => setVal(e.target.value)}
                        className="px-3 py-1.5 rounded-lg bg-surface-base border border-border-default text-text-primary text-sm focus:outline-none focus:border-accent-cyan"
                    >
                        {options.map((o) => (
                            <option key={o.value} value={o.value}>{o.label}</option>
                        ))}
                    </select>
                ) : (
                    <input
                        type={type}
                        value={val}
                        onChange={(e) => setVal(e.target.value)}
                        className="w-24 px-3 py-1.5 rounded-lg bg-surface-base border border-border-default text-text-primary text-sm font-mono focus:outline-none focus:border-accent-cyan"
                    />
                )}
                {changed && (
                    <button
                        onClick={() => onSave(val)}
                        className="px-3 py-1.5 rounded-lg bg-accent-cyan/15 text-accent-cyan text-xs font-medium hover:bg-accent-cyan/25 transition-colors"
                    >
                        Save
                    </button>
                )}
            </div>
        </div>
    );
}
