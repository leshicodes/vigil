import { useState } from 'react';
import type { Schedule } from '../lib/api';

interface ScheduleFormProps {
    initial?: Schedule;
    onSubmit: (data: Omit<Schedule, 'id'>) => void;
    onCancel: () => void;
}

const cameraDrivers = [
    { value: 'mock', label: 'Mock (Test)' },
    { value: 'libcamera', label: 'libcamera (RPi CSI)' },
    { value: 'fswebcam', label: 'fswebcam (USB)' },
    { value: 'ffmpeg', label: 'ffmpeg (USB/DirectShow)' },
];

const presets = [
    { label: 'Every 5 min', value: '*/5 * * * *' },
    { label: 'Every 15 min', value: '*/15 * * * *' },
    { label: 'Every 30 min', value: '*/30 * * * *' },
    { label: 'Every hour', value: '0 * * * *' },
    { label: 'Every 6 hours', value: '0 */6 * * *' },
    { label: 'Daily at noon', value: '0 12 * * *' },
];

export default function ScheduleForm({ initial, onSubmit, onCancel }: ScheduleFormProps) {
    const [cronExpr, setCronExpr] = useState(initial?.cron_expr ?? '*/30 * * * *');
    const [cameraId, setCameraId] = useState(initial?.camera_id ?? 'mock');
    const [hookPath, setHookPath] = useState(initial?.hook_path ?? '');
    const [enabled, setEnabled] = useState(initial?.enabled ?? true);

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        onSubmit({ cron_expr: cronExpr, camera_id: cameraId, hook_path: hookPath, enabled });
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-5 p-6">
            {/* Cron Expression */}
            <div>
                <label className="block text-sm font-medium text-text-primary mb-2">Schedule (Cron)</label>
                <input
                    type="text"
                    value={cronExpr}
                    onChange={(e) => setCronExpr(e.target.value)}
                    className="w-full px-3 py-2.5 rounded-lg bg-surface-base border border-border-default text-text-primary text-sm font-mono focus:outline-none focus:border-accent-cyan focus:ring-1 focus:ring-accent-cyan/30 transition-colors"
                    placeholder="*/30 * * * *"
                />
                <div className="flex flex-wrap gap-1.5 mt-2">
                    {presets.map((p) => (
                        <button
                            key={p.value}
                            type="button"
                            onClick={() => setCronExpr(p.value)}
                            className={`px-2 py-1 rounded-md text-[11px] font-medium transition-colors ${cronExpr === p.value
                                    ? 'bg-accent-cyan/15 text-accent-cyan border border-accent-cyan/30'
                                    : 'bg-surface-hover text-text-muted border border-transparent hover:text-text-secondary'
                                }`}
                        >
                            {p.label}
                        </button>
                    ))}
                </div>
            </div>

            {/* Camera Driver */}
            <div>
                <label className="block text-sm font-medium text-text-primary mb-2">Camera Driver</label>
                <select
                    value={cameraId}
                    onChange={(e) => setCameraId(e.target.value)}
                    className="w-full px-3 py-2.5 rounded-lg bg-surface-base border border-border-default text-text-primary text-sm focus:outline-none focus:border-accent-cyan focus:ring-1 focus:ring-accent-cyan/30 transition-colors"
                >
                    {cameraDrivers.map((d) => (
                        <option key={d.value} value={d.value}>{d.label}</option>
                    ))}
                </select>
            </div>

            {/* Hook Path */}
            <div>
                <label className="block text-sm font-medium text-text-primary mb-2">
                    Hook Script <span className="text-text-muted font-normal">(optional)</span>
                </label>
                <input
                    type="text"
                    value={hookPath}
                    onChange={(e) => setHookPath(e.target.value)}
                    className="w-full px-3 py-2.5 rounded-lg bg-surface-base border border-border-default text-text-primary text-sm font-mono focus:outline-none focus:border-accent-cyan focus:ring-1 focus:ring-accent-cyan/30 transition-colors"
                    placeholder="/hooks/analyze.sh"
                />
            </div>

            {/* Enabled Toggle */}
            <div className="flex items-center gap-3">
                <button
                    type="button"
                    onClick={() => setEnabled(!enabled)}
                    className={`relative w-10 h-5.5 rounded-full transition-colors duration-200 ${enabled ? 'bg-accent-cyan' : 'bg-surface-hover border border-border-default'
                        }`}
                >
                    <span className={`absolute top-0.5 left-0.5 w-4.5 h-4.5 rounded-full bg-white shadow transition-transform duration-200 ${enabled ? 'translate-x-[18px]' : ''
                        }`} />
                </button>
                <span className="text-sm text-text-secondary">
                    {enabled ? 'Enabled' : 'Disabled'}
                </span>
            </div>

            {/* Actions */}
            <div className="flex gap-3 pt-2">
                <button
                    type="submit"
                    className="flex-1 px-4 py-2.5 rounded-lg bg-gradient-to-r from-accent-cyan to-accent-teal text-surface-base text-sm font-semibold hover:opacity-90 transition-opacity"
                >
                    {initial ? 'Update Schedule' : 'Create Schedule'}
                </button>
                <button
                    type="button"
                    onClick={onCancel}
                    className="px-4 py-2.5 rounded-lg border border-border-default text-text-secondary text-sm hover:bg-surface-hover transition-colors"
                >
                    Cancel
                </button>
            </div>
        </form>
    );
}
