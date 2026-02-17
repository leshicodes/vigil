interface StatusBadgeProps {
    status: 'ok' | 'warn' | 'error' | 'offline';
    label?: string;
    size?: 'sm' | 'md';
}

const colors = {
    ok: 'bg-status-ok',
    warn: 'bg-status-warn',
    error: 'bg-status-error',
    offline: 'bg-text-muted',
};

export default function StatusBadge({ status, label, size = 'md' }: StatusBadgeProps) {
    const dotSize = size === 'sm' ? 'w-2 h-2' : 'w-2.5 h-2.5';

    return (
        <span className="inline-flex items-center gap-2">
            <span className={`relative flex ${dotSize}`}>
                <span className={`absolute inline-flex h-full w-full rounded-full ${colors[status]} opacity-40 ${status === 'ok' ? 'animate-ping' : ''}`} />
                <span className={`relative inline-flex rounded-full ${dotSize} ${colors[status]}`} />
            </span>
            {label && (
                <span className={`font-medium ${size === 'sm' ? 'text-xs' : 'text-sm'} text-text-secondary`}>
                    {label}
                </span>
            )}
        </span>
    );
}
