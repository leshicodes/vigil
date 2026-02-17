import { useEffect, useState, useRef, useCallback } from 'react';
import { ChevronLeft, ChevronRight, Play, Pause, SkipBack, SkipForward, Trash2, CheckSquare, Square, XCircle } from 'lucide-react';
import { api, type CaptureLog } from '../lib/api';
import { toast } from '../components/Toast';

export default function Timeline() {
    const [date, setDate] = useState(() => new Date().toISOString().slice(0, 10));
    const [captures, setCaptures] = useState<CaptureLog[]>([]);
    const [loading, setLoading] = useState(true);
    const [selectedIdx, setSelectedIdx] = useState<number | null>(null);
    const [playing, setPlaying] = useState(false);
    const [speed, setSpeed] = useState(1000);
    const playRef = useRef<ReturnType<typeof setInterval> | null>(null);

    // Selection state
    const [selectMode, setSelectMode] = useState(false);
    const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
    const [deleting, setDeleting] = useState(false);

    const fetchCaptures = useCallback(async () => {
        setLoading(true);
        try {
            const caps = await api.getCaptures(date, 500);
            setCaptures(caps.reverse());
        } catch (err) {
            console.error('Timeline fetch error:', err);
            setCaptures([]);
        }
        setLoading(false);
    }, [date]);

    useEffect(() => {
        fetchCaptures();
    }, [fetchCaptures]);

    // Playback
    useEffect(() => {
        if (playing && captures.length > 0) {
            playRef.current = setInterval(() => {
                setSelectedIdx((prev) => {
                    const next = (prev ?? -1) + 1;
                    if (next >= captures.length) {
                        setPlaying(false);
                        return prev;
                    }
                    return next;
                });
            }, speed);
        }
        return () => {
            if (playRef.current) clearInterval(playRef.current);
        };
    }, [playing, speed, captures.length]);

    const shiftDate = (days: number) => {
        const d = new Date(date + 'T00:00:00');
        d.setDate(d.getDate() + days);
        setDate(d.toISOString().slice(0, 10));
        setSelectedIdx(null);
        setPlaying(false);
        exitSelectMode();
    };

    const startPlayback = () => {
        if (captures.length === 0) return;
        setSelectedIdx(0);
        setPlaying(true);
    };

    const exitSelectMode = () => {
        setSelectMode(false);
        setSelectedIds(new Set());
    };

    const toggleSelect = (id: number) => {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id);
            else next.add(id);
            return next;
        });
    };

    const selectAll = () => {
        setSelectedIds(new Set(captures.map((c) => c.id)));
    };

    const selectNone = () => {
        setSelectedIds(new Set());
    };

    const handleDelete = async () => {
        if (selectedIds.size === 0) return;
        const count = selectedIds.size;
        setDeleting(true);
        try {
            await api.deleteCaptures(Array.from(selectedIds));
            toast('success', `Deleted ${count} capture${count !== 1 ? 's' : ''}`);
            exitSelectMode();
            setSelectedIdx(null);
            fetchCaptures();
        } catch (err) {
            toast('error', `Delete failed: ${err}`);
        }
        setDeleting(false);
    };

    const currentCapture = selectedIdx !== null ? captures[selectedIdx] : null;

    const speeds = [
        { label: '0.5×', value: 2000 },
        { label: '1×', value: 1000 },
        { label: '2×', value: 500 },
        { label: '4×', value: 250 },
    ];

    return (
        <div className="space-y-6 animate-fade-in">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-text-primary">Timeline</h1>
                    <p className="text-sm text-text-muted mt-1">Browse and replay your captures</p>
                </div>
                {captures.length > 0 && !selectMode && (
                    <button
                        onClick={() => setSelectMode(true)}
                        className="flex items-center gap-2 px-3 py-2 rounded-lg border border-border-default text-text-secondary text-sm hover:bg-surface-hover transition-colors"
                    >
                        <CheckSquare className="w-4 h-4" />
                        Select
                    </button>
                )}
            </div>

            {/* Date picker */}
            <div className="flex items-center gap-3">
                <button
                    onClick={() => shiftDate(-1)}
                    className="p-2 rounded-lg border border-border-default text-text-secondary hover:bg-surface-hover hover:text-text-primary transition-colors"
                >
                    <ChevronLeft className="w-4 h-4" />
                </button>

                <input
                    type="date"
                    value={date}
                    onChange={(e) => { setDate(e.target.value); setSelectedIdx(null); setPlaying(false); exitSelectMode(); }}
                    className="px-4 py-2 rounded-lg bg-surface-card border border-border-default text-text-primary text-sm focus:outline-none focus:border-accent-cyan"
                />

                <button
                    onClick={() => shiftDate(1)}
                    className="p-2 rounded-lg border border-border-default text-text-secondary hover:bg-surface-hover hover:text-text-primary transition-colors"
                >
                    <ChevronRight className="w-4 h-4" />
                </button>

                <span className="text-sm text-text-muted ml-2">
                    {captures.length} capture{captures.length !== 1 ? 's' : ''}
                </span>
            </div>

            {/* Selection toolbar */}
            {selectMode && (
                <div className="glass rounded-xl px-5 py-3 flex items-center gap-3 flex-wrap">
                    <span className="text-sm font-medium text-text-primary">
                        {selectedIds.size} selected
                    </span>

                    <div className="flex items-center gap-1.5 ml-2">
                        <button
                            onClick={selectAll}
                            className="px-2.5 py-1.5 rounded-lg text-xs font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors"
                        >
                            Select All
                        </button>
                        <button
                            onClick={selectNone}
                            className="px-2.5 py-1.5 rounded-lg text-xs font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors"
                        >
                            Select None
                        </button>
                    </div>

                    <div className="flex items-center gap-2 ml-auto">
                        <button
                            onClick={handleDelete}
                            disabled={selectedIds.size === 0 || deleting}
                            className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-status-error/15 text-status-error text-sm font-medium hover:bg-status-error/25 transition-colors disabled:opacity-40"
                        >
                            <Trash2 className="w-3.5 h-3.5" />
                            {deleting ? 'Deleting...' : `Delete ${selectedIds.size > 0 ? `(${selectedIds.size})` : ''}`}
                        </button>
                        <button
                            onClick={exitSelectMode}
                            className="p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
                            title="Exit selection"
                        >
                            <XCircle className="w-4.5 h-4.5" />
                        </button>
                    </div>
                </div>
            )}

            {/* Playback controls (hidden in select mode) */}
            {captures.length > 0 && !selectMode && (
                <div className="glass rounded-xl px-5 py-3 flex items-center gap-4 flex-wrap">
                    <div className="flex items-center gap-2">
                        <button
                            onClick={() => setSelectedIdx(0)}
                            className="p-1.5 rounded-lg text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors"
                            title="First frame"
                        >
                            <SkipBack className="w-4 h-4" />
                        </button>

                        <button
                            onClick={() => setSelectedIdx((prev) => Math.max(0, (prev ?? 0) - 1))}
                            className="p-1.5 rounded-lg text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors"
                        >
                            <ChevronLeft className="w-4 h-4" />
                        </button>

                        <button
                            onClick={() => playing ? setPlaying(false) : startPlayback()}
                            className="p-2.5 rounded-xl bg-gradient-to-r from-accent-cyan to-accent-teal text-surface-base hover:opacity-90 transition-opacity"
                        >
                            {playing ? <Pause className="w-5 h-5" /> : <Play className="w-5 h-5" />}
                        </button>

                        <button
                            onClick={() => setSelectedIdx((prev) => Math.min(captures.length - 1, (prev ?? 0) + 1))}
                            className="p-1.5 rounded-lg text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors"
                        >
                            <ChevronRight className="w-4 h-4" />
                        </button>

                        <button
                            onClick={() => setSelectedIdx(captures.length - 1)}
                            className="p-1.5 rounded-lg text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors"
                            title="Last frame"
                        >
                            <SkipForward className="w-4 h-4" />
                        </button>
                    </div>

                    {/* Frame counter */}
                    <span className="text-xs text-text-muted font-mono">
                        {selectedIdx !== null ? selectedIdx + 1 : '—'} / {captures.length}
                    </span>

                    {/* Speed controls */}
                    <div className="flex items-center gap-1 ml-auto">
                        {speeds.map((s) => (
                            <button
                                key={s.value}
                                onClick={() => setSpeed(s.value)}
                                className={`px-2 py-1 rounded-md text-xs font-medium transition-colors ${speed === s.value
                                    ? 'bg-accent-cyan/15 text-accent-cyan'
                                    : 'text-text-muted hover:text-text-secondary'
                                    }`}
                            >
                                {s.label}
                            </button>
                        ))}
                    </div>

                    {/* Scrubber */}
                    <input
                        type="range"
                        min={0}
                        max={Math.max(0, captures.length - 1)}
                        value={selectedIdx ?? 0}
                        onChange={(e) => { setSelectedIdx(Number(e.target.value)); setPlaying(false); }}
                        className="w-full mt-1 accent-accent-cyan"
                    />
                </div>
            )}

            {/* Playback viewer (hidden in select mode) */}
            {currentCapture && !selectMode && (
                <div className="rounded-2xl overflow-hidden border border-border-subtle glow-cyan">
                    <div className="aspect-video bg-black flex items-center justify-center">
                        <img
                            src={api.getCaptureUrl(currentCapture.filepath)}
                            alt={`Frame ${selectedIdx! + 1}`}
                            className="max-w-full max-h-full object-contain"
                        />
                    </div>
                    <div className="glass-strong px-4 py-2 flex items-center justify-between text-xs">
                        <span className="text-text-secondary">
                            {new Date(currentCapture.timestamp).toLocaleString()}
                        </span>
                        <div className="flex items-center gap-2">
                            {(() => {
                                const a = parseAnalysis(currentCapture.hook_output);
                                if (!a) return null;
                                return (
                                    <>
                                        {a.motion_detected && (
                                            <span className="px-1.5 py-0.5 rounded bg-status-warning/20 text-status-warning text-[10px] font-bold">
                                                MOTION
                                            </span>
                                        )}
                                        <span className="text-text-muted font-mono text-[10px]">
                                            Brightness: {a.brightness} · Diff: {a.change_pct}%
                                        </span>
                                    </>
                                );
                            })()}
                            <span className="text-text-muted font-mono">
                                {currentCapture.filepath.split(/[\\/]/).pop()}
                            </span>
                        </div>
                    </div>
                </div>
            )}

            {/* Image Grid */}
            {!loading && captures.length > 0 && (
                <div>
                    <h2 className="text-lg font-semibold text-text-primary mb-4">
                        {selectMode ? 'Select captures to delete' : 'All Captures'}
                    </h2>
                    <div className="grid grid-cols-3 sm:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8 gap-2">
                        {captures.map((cap, idx) => {
                            const isSelected = selectMode && selectedIds.has(cap.id);
                            const isViewing = !selectMode && selectedIdx === idx;
                            const hasMotion = hasMotionDetected(cap.hook_output);

                            return (
                                <button
                                    key={cap.id}
                                    onClick={() => {
                                        if (selectMode) {
                                            toggleSelect(cap.id);
                                        } else {
                                            setSelectedIdx(idx);
                                            setPlaying(false);
                                        }
                                    }}
                                    className={`group relative aspect-video rounded-lg overflow-hidden border transition-all duration-200 cursor-pointer ${isViewing
                                        ? 'border-accent-cyan glow-cyan ring-2 ring-accent-cyan/30'
                                        : isSelected
                                            ? 'border-status-error ring-2 ring-status-error/30'
                                            : 'border-border-subtle hover:border-border-bright'
                                        }`}
                                >
                                    <img
                                        src={api.getCaptureUrl(cap.filepath)}
                                        alt={`Capture ${idx + 1}`}
                                        className={`w-full h-full object-cover transition-opacity ${isSelected ? 'opacity-50' : ''}`}
                                        loading="lazy"
                                    />

                                    {/* Selection checkbox */}
                                    {selectMode && (
                                        <div className="absolute top-1 left-1">
                                            {isSelected ? (
                                                <CheckSquare className="w-5 h-5 text-status-error drop-shadow" />
                                            ) : (
                                                <Square className="w-5 h-5 text-white/70 drop-shadow" />
                                            )}
                                        </div>
                                    )}

                                    <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent px-1.5 py-1 flex items-center justify-between">
                                        <p className="text-[9px] text-white/80 font-mono">
                                            {new Date(cap.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                        </p>
                                        {hasMotion && (
                                            <span className="w-2 h-2 rounded-full bg-status-warning animate-pulse" title="Motion detected" />
                                        )}
                                    </div>
                                </button>
                            );
                        })}
                    </div>
                </div>
            )}

            {/* Empty state */}
            {!loading && captures.length === 0 && (
                <div className="rounded-2xl border border-border-subtle bg-surface-card flex items-center justify-center py-16">
                    <div className="text-center">
                        <p className="text-text-secondary font-medium">No captures for {date}</p>
                        <p className="text-sm text-text-muted mt-1">Try selecting a different date</p>
                    </div>
                </div>
            )}

            {loading && (
                <div className="flex items-center justify-center py-16">
                    <div className="w-8 h-8 rounded-full border-2 border-accent-cyan border-t-transparent animate-spin" />
                </div>
            )}
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
        const firstLine = hookOutput.trim().split('\n')[0];
        const data = JSON.parse(firstLine);
        if (typeof data.brightness === 'number') return data as AnalysisData;
        return null;
    } catch {
        return null;
    }
}

function hasMotionDetected(hookOutput: string): boolean {
    const a = parseAnalysis(hookOutput);
    return a?.motion_detected === true;
}
