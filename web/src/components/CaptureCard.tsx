import { api, handleImageError } from '../lib/api';

interface CaptureCardProps {
    filepath: string;
    timestamp: string;
    hookStatus: string;
    onClick?: () => void;
}

export default function CaptureCard({ filepath, timestamp, hookStatus, onClick }: CaptureCardProps) {
    const imgUrl = api.getCaptureUrl(filepath);
    const time = new Date(timestamp);
    const timeStr = time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    const dateStr = time.toLocaleDateString([], { month: 'short', day: 'numeric' });

    return (
        <button
            onClick={onClick}
            className="group relative rounded-xl overflow-hidden border border-border-subtle bg-surface-card hover:border-accent-cyan/40 transition-all duration-300 hover:glow-cyan cursor-pointer focus:outline-none focus:ring-2 focus:ring-accent-cyan/50"
        >
            <div className="aspect-video bg-surface-base">
                <img
                    src={imgUrl}
                    alt={`Capture at ${timeStr}`}
                    className="w-full h-full object-cover opacity-90 group-hover:opacity-100 transition-opacity duration-300"
                    loading="lazy"
                    onError={handleImageError}
                />
            </div>

            {/* Overlay gradient */}
            <div className="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-black/80 to-transparent" />

            {/* Info */}
            <div className="absolute bottom-0 inset-x-0 px-3 py-2 flex items-center justify-between">
                <div>
                    <p className="text-sm font-medium text-white">{timeStr}</p>
                    <p className="text-[10px] text-white/60">{dateStr}</p>
                </div>
                {hookStatus !== 'skipped' && (
                    <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full ${hookStatus === 'success'
                        ? 'bg-status-ok/20 text-status-ok'
                        : 'bg-status-error/20 text-status-error'
                        }`}>
                        {hookStatus}
                    </span>
                )}
            </div>
        </button>
    );
}
