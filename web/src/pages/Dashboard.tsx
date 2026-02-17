import { useEffect, useState } from 'react';
import { Camera, Clock, CalendarDays, Activity, Aperture } from 'lucide-react';
import { api, type CaptureLog, type Status } from '../lib/api';
import CaptureCard from '../components/CaptureCard';
import StatusBadge from '../components/StatusBadge';
import Modal from '../components/Modal';
import { toast } from '../components/Toast';

export default function Dashboard() {
    const [status, setStatus] = useState<Status | null>(null);
    const [captures, setCaptures] = useState<CaptureLog[]>([]);
    const [scheduleCount, setScheduleCount] = useState(0);
    const [selectedCapture, setSelectedCapture] = useState<CaptureLog | null>(null);
    const [triggering, setTriggering] = useState(false);

    const fetchData = async () => {
        try {
            const [s, caps, scheds] = await Promise.all([
                api.getStatus(),
                api.getCaptures(undefined, 12),
                api.getSchedules(),
            ]);
            setStatus(s);
            setCaptures(caps);
            setScheduleCount(scheds.filter((s) => s.enabled).length);
        } catch (err) {
            console.error('Dashboard fetch error:', err);
        }
    };

    useEffect(() => {
        fetchData();
        const id = setInterval(fetchData, 30_000);
        return () => clearInterval(id);
    }, []);

    const latest = captures[0];
    const todayStr = new Date().toISOString().slice(0, 10);
    const todayCaptures = captures.filter((c) => c.timestamp.startsWith(todayStr));

    return (
        <div className="space-y-8 animate-fade-in">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-text-primary">Dashboard</h1>
                    <p className="text-sm text-text-muted mt-1">Your time-lapse monitoring overview</p>
                </div>
                <div className="flex items-center gap-3">
                    <button
                        onClick={async () => {
                            setTriggering(true);
                            try {
                                const result = await api.triggerCapture();
                                toast('success', `Capture triggered (${result.camera_id})`);
                                setTimeout(() => fetchData(), 3000);
                            } catch (err) {
                                toast('error', `Trigger failed: ${err}`);
                            }
                            setTriggering(false);
                        }}
                        disabled={triggering}
                        className="flex items-center gap-2 px-4 py-2 rounded-lg bg-gradient-to-r from-accent-cyan to-accent-teal text-surface-base text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-50"
                    >
                        <Aperture className={`w-4 h-4 ${triggering ? 'animate-spin' : ''}`} />
                        {triggering ? 'Capturing...' : 'Capture Now'}
                    </button>
                    <StatusBadge
                        status={status?.status === 'ok' ? 'ok' : 'offline'}
                        label={status?.status === 'ok' ? 'Online' : 'Offline'}
                    />
                </div>
            </div>

            {/* Stats Row */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard icon={Activity} label="Uptime" value={status?.uptime ?? '—'} color="cyan" />
                <StatCard icon={Camera} label="Today" value={`${todayCaptures.length} captures`} color="teal" />
                <StatCard icon={Clock} label="Schedules" value={`${scheduleCount} active`} color="emerald" />
                <StatCard
                    icon={CalendarDays}
                    label="Last Capture"
                    value={latest ? new Date(latest.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '—'}
                    color="blue"
                />
            </div>

            {/* Latest Capture Hero */}
            {latest && (() => {
                const analysis = parseAnalysis(latest.hook_output);
                return (
                    <div className="relative rounded-2xl overflow-hidden border border-border-subtle glow-cyan">
                        <div className="aspect-video max-h-96 bg-surface-base">
                            <img
                                src={api.getCaptureUrl(latest.filepath)}
                                alt="Latest capture"
                                className="w-full h-full object-contain bg-black"
                            />
                        </div>
                        <div className="absolute inset-x-0 bottom-0 glass-strong px-6 py-4">
                            <div className="flex items-center justify-between">
                                <div>
                                    <p className="text-sm font-medium text-text-primary">Latest Capture</p>
                                    <p className="text-xs text-text-muted">{new Date(latest.timestamp).toLocaleString()}</p>
                                </div>
                                <div className="flex items-center gap-2">
                                    {analysis && (
                                        <>
                                            {analysis.motion_detected && (
                                                <span className="px-2 py-0.5 rounded-full bg-status-warning/20 text-status-warning text-[10px] font-bold uppercase tracking-wider">
                                                    Motion
                                                </span>
                                            )}
                                            <span className="text-[10px] text-text-muted font-mono">
                                                🔅{analysis.brightness} · 🔍{analysis.sharpness} · Δ{analysis.change_pct}%
                                            </span>
                                        </>
                                    )}
                                    <StatusBadge
                                        status={latest.hook_status === 'success' ? 'ok' : latest.hook_status === 'error' ? 'error' : 'offline'}
                                        label={`Hook: ${latest.hook_status}`}
                                        size="sm"
                                    />
                                </div>
                            </div>
                        </div>
                    </div>
                );
            })()}

            {!latest && (
                <div className="rounded-2xl border border-border-subtle bg-surface-card flex items-center justify-center py-20">
                    <div className="text-center">
                        <Camera className="w-12 h-12 text-text-muted mx-auto mb-3" />
                        <p className="text-text-secondary font-medium">No captures yet</p>
                        <p className="text-sm text-text-muted mt-1">Start the engine to begin capturing</p>
                    </div>
                </div>
            )}

            {/* Recent Captures */}
            {captures.length > 1 && (
                <div>
                    <h2 className="text-lg font-semibold text-text-primary mb-4">Recent Captures</h2>
                    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-3">
                        {captures.slice(1).map((cap) => (
                            <CaptureCard
                                key={cap.id}
                                filepath={cap.filepath}
                                timestamp={cap.timestamp}
                                hookStatus={cap.hook_status}
                                onClick={() => setSelectedCapture(cap)}
                            />
                        ))}
                    </div>
                </div>
            )}

            {/* Lightbox Modal */}
            <Modal open={!!selectedCapture} onClose={() => setSelectedCapture(null)}>
                {selectedCapture && (
                    <div>
                        <img
                            src={api.getCaptureUrl(selectedCapture.filepath)}
                            alt="Capture"
                            className="w-full"
                        />
                        <div className="px-6 py-4 border-t border-border-subtle">
                            <p className="text-sm text-text-primary font-medium">
                                {new Date(selectedCapture.timestamp).toLocaleString()}
                            </p>
                            <p className="text-xs text-text-muted mt-1 font-mono">{selectedCapture.filepath}</p>
                            {selectedCapture.hook_output && (() => {
                                const a = parseAnalysis(selectedCapture.hook_output);
                                if (a) {
                                    return (
                                        <div className="mt-3 grid grid-cols-2 sm:grid-cols-4 gap-2">
                                            <MetricPill label="Brightness" value={String(a.brightness)} />
                                            <MetricPill label="Sharpness" value={String(a.sharpness)} />
                                            <MetricPill label="Change" value={`${a.change_pct}%`} />
                                            <MetricPill label="Motion" value={a.motion_detected ? 'Yes' : 'No'}
                                                highlight={a.motion_detected} />
                                            <MetricPill label="Resolution" value={a.resolution} />
                                            <MetricPill label="Size" value={`${a.file_size_kb} KB`} />
                                        </div>
                                    );
                                }
                                return (
                                    <pre className="mt-3 p-3 rounded-lg bg-surface-base text-xs text-text-secondary overflow-x-auto font-mono">
                                        {selectedCapture.hook_output}
                                    </pre>
                                );
                            })()}
                        </div>
                    </div>
                )}
            </Modal>
        </div>
    );
}

// --- Helpers ----------------------------------------------------------------

interface AnalysisData {
    brightness: number;
    sharpness: number;
    resolution: string;
    file_size_kb: number;
    change_pct: number;
    motion_detected: boolean;
    timestamp: string;
}

function parseAnalysis(hookOutput: string): AnalysisData | null {
    if (!hookOutput) return null;
    try {
        // The hook output may contain stderr after the JSON — take only the first line.
        const firstLine = hookOutput.trim().split('\n')[0];
        const data = JSON.parse(firstLine);
        if (typeof data.brightness === 'number') return data as AnalysisData;
        return null;
    } catch {
        return null;
    }
}

function MetricPill({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
    return (
        <div className={`px-3 py-1.5 rounded-lg text-xs font-mono ${highlight ? 'bg-status-warning/15 text-status-warning' : 'bg-surface-base text-text-secondary'
            }`}>
            <span className="text-text-muted">{label}:</span>{' '}
            <span className="font-semibold">{value}</span>
        </div>
    );
}

// --- Helper Components ---

function StatCard({ icon: Icon, label, value, color }: {
    icon: React.ComponentType<{ className?: string }>;
    label: string;
    value: string;
    color: 'cyan' | 'teal' | 'emerald' | 'blue';
}) {
    const iconColors = {
        cyan: 'text-accent-cyan bg-accent-cyan/10',
        teal: 'text-accent-teal bg-accent-teal/10',
        emerald: 'text-accent-emerald bg-accent-emerald/10',
        blue: 'text-accent-blue bg-accent-blue/10',
    };

    return (
        <div className="glass rounded-xl p-4 hover:border-border-bright transition-colors">
            <div className="flex items-center gap-3">
                <div className={`p-2 rounded-lg ${iconColors[color]}`}>
                    <Icon className="w-4.5 h-4.5" />
                </div>
                <div className="min-w-0">
                    <p className="text-xs text-text-muted">{label}</p>
                    <p className="text-sm font-semibold text-text-primary truncate mt-0.5">{value}</p>
                </div>
            </div>
        </div>
    );
}
