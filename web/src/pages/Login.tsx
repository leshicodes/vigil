import { useState, type FormEvent } from 'react';
import { Eye, EyeOff, Lock } from 'lucide-react';
import { api, setToken } from '../lib/api';

interface LoginProps {
    onSuccess: () => void;
}

export default function Login({ onSuccess }: LoginProps) {
    const [key, setKey] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [showKey, setShowKey] = useState(false);

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        if (!key.trim()) return;

        setLoading(true);
        setError('');

        try {
            const res = await api.login(key.trim());
            if (res.ok) {
                setToken(key.trim());
                onSuccess();
            }
        } catch {
            setError('Invalid API key');
        }
        setLoading(false);
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-surface-base p-4">
            <div className="w-full max-w-sm">
                {/* Logo */}
                <div className="text-center mb-8">
                    <pre className="text-accent-cyan text-sm font-mono inline-block text-left leading-tight">
                        {` _  _ _ ____ _ _   
 \\/ |/ |  __|| | |  
  \\   /| |__ | | |  
   | | |  __|| | |  
   | | | |__ | | |__
   |_| |____||_|____|`}
                    </pre>
                    <p className="text-text-muted text-sm mt-3">Time-Lapse Monitor</p>
                </div>

                {/* Login form */}
                <form onSubmit={handleSubmit} className="glass rounded-2xl p-6 space-y-4">
                    <div className="flex items-center gap-2 mb-2">
                        <Lock className="w-4 h-4 text-accent-cyan" />
                        <h2 className="text-lg font-semibold text-text-primary">Authentication</h2>
                    </div>

                    <p className="text-sm text-text-muted">
                        Enter your API key to continue.
                    </p>

                    <div className="relative">
                        <input
                            type={showKey ? 'text' : 'password'}
                            value={key}
                            onChange={(e) => setKey(e.target.value)}
                            placeholder="API key"
                            autoFocus
                            className="w-full px-4 py-2.5 pr-10 rounded-lg bg-surface-base border border-border-default text-text-primary text-sm placeholder:text-text-muted focus:outline-none focus:border-accent-cyan transition-colors"
                        />
                        <button
                            type="button"
                            onClick={() => setShowKey(!showKey)}
                            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
                        >
                            {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </button>
                    </div>

                    {error && (
                        <p className="text-sm text-status-error">{error}</p>
                    )}

                    <button
                        type="submit"
                        disabled={loading || !key.trim()}
                        className="w-full py-2.5 rounded-lg bg-gradient-to-r from-accent-cyan to-accent-teal text-surface-base font-medium text-sm hover:opacity-90 transition-opacity disabled:opacity-40"
                    >
                        {loading ? 'Checking...' : 'Unlock'}
                    </button>
                </form>
            </div>
        </div>
    );
}
