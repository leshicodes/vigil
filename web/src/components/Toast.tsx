import { useEffect, useState } from 'react';
import { CheckCircle, XCircle, X } from 'lucide-react';

export type ToastType = 'success' | 'error';

interface Toast {
    id: number;
    type: ToastType;
    message: string;
}

let toastId = 0;
let addToastFn: ((type: ToastType, message: string) => void) | null = null;

export function toast(type: ToastType, message: string) {
    addToastFn?.(type, message);
}

export default function ToastContainer() {
    const [toasts, setToasts] = useState<Toast[]>([]);

    useEffect(() => {
        addToastFn = (type, message) => {
            const id = ++toastId;
            setToasts((prev) => [...prev, { id, type, message }]);
            setTimeout(() => {
                setToasts((prev) => prev.filter((t) => t.id !== id));
            }, 4000);
        };
        return () => { addToastFn = null; };
    }, []);

    const dismiss = (id: number) => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
    };

    return (
        <div className="fixed bottom-4 right-4 z-[60] flex flex-col gap-2 max-w-sm">
            {toasts.map((t) => (
                <div
                    key={t.id}
                    className={`glass-strong rounded-xl px-4 py-3 flex items-start gap-3 animate-fade-in shadow-lg ${t.type === 'success' ? 'border-status-ok/30' : 'border-status-error/30'
                        }`}
                >
                    {t.type === 'success' ? (
                        <CheckCircle className="w-5 h-5 text-status-ok shrink-0 mt-0.5" />
                    ) : (
                        <XCircle className="w-5 h-5 text-status-error shrink-0 mt-0.5" />
                    )}
                    <p className="text-sm text-text-primary flex-1">{t.message}</p>
                    <button onClick={() => dismiss(t.id)} className="text-text-muted hover:text-text-primary">
                        <X className="w-4 h-4" />
                    </button>
                </div>
            ))}
        </div>
    );
}
