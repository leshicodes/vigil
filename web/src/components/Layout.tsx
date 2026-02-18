import { useState, useEffect } from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { LayoutDashboard, Clock, Settings, Eye, Wifi, WifiOff, LogOut } from 'lucide-react';
import { api, clearToken } from '../lib/api';

const navItems = [
    { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
    { to: '/timeline', icon: Clock, label: 'Timeline' },
    { to: '/config', icon: Settings, label: 'Config' },
];

interface LayoutProps {
    onLogout?: () => void;
}

export default function Layout({ onLogout }: LayoutProps) {
    const [online, setOnline] = useState(false);
    const [uptime, setUptime] = useState('');

    useEffect(() => {
        const check = async () => {
            try {
                const s = await api.getStatus();
                setOnline(s.status === 'ok');
                setUptime(s.uptime);
            } catch {
                setOnline(false);
            }
        };
        check();
        const id = setInterval(check, 10_000);
        return () => clearInterval(id);
    }, []);

    const handleLogout = async () => {
        try {
            await api.logout();
        } catch {
            // Server might be unreachable — clear locally anyway.
        }
        clearToken();
        if (onLogout) onLogout();
    };

    return (
        <div className="flex h-screen overflow-hidden">
            {/* Sidebar */}
            <aside className="hidden md:flex flex-col w-64 border-r border-border-subtle bg-surface-card">
                {/* Logo */}
                <div className="flex items-center gap-3 px-6 py-5 border-b border-border-subtle">
                    <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-accent-cyan to-accent-teal flex items-center justify-center">
                        <Eye className="w-5 h-5 text-surface-base" />
                    </div>
                    <div>
                        <h1 className="text-lg font-semibold tracking-tight text-text-primary">Vigil</h1>
                        <p className="text-xs text-text-muted">Time-Lapse Monitor</p>
                    </div>
                </div>

                {/* Nav */}
                <nav className="flex-1 px-3 py-4 space-y-1">
                    {navItems.map(({ to, icon: Icon, label }) => (
                        <NavLink
                            key={to}
                            to={to}
                            end={to === '/'}
                            className={({ isActive }) =>
                                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 ${isActive
                                    ? 'bg-accent-cyan/10 text-accent-cyan glow-cyan'
                                    : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'
                                }`
                            }
                        >
                            <Icon className="w-4.5 h-4.5" />
                            {label}
                        </NavLink>
                    ))}
                </nav>

                {/* Status footer */}
                <div className="px-4 py-4 border-t border-border-subtle space-y-3">
                    <div className="glass rounded-lg px-3 py-2.5 flex items-center gap-2.5">
                        <div className="relative">
                            {online ? (
                                <Wifi className="w-4 h-4 text-status-ok" />
                            ) : (
                                <WifiOff className="w-4 h-4 text-status-error" />
                            )}
                        </div>
                        <div className="min-w-0">
                            <p className="text-xs font-medium text-text-primary truncate">
                                {online ? 'Engine Online' : 'Engine Offline'}
                            </p>
                            {online && uptime && (
                                <p className="text-[10px] text-text-muted truncate">Up {uptime}</p>
                            )}
                        </div>
                    </div>

                    {onLogout && (
                        <button
                            onClick={handleLogout}
                            className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-text-muted hover:text-status-error hover:bg-status-error/10 transition-colors"
                        >
                            <LogOut className="w-4 h-4" />
                            Logout
                        </button>
                    )}
                </div>
            </aside>

            {/* Main content */}
            <main className="flex-1 overflow-y-auto">
                <div className="p-6 lg:p-8 max-w-7xl mx-auto">
                    <Outlet />
                </div>
            </main>

            {/* Mobile bottom nav */}
            <nav className="md:hidden fixed bottom-0 left-0 right-0 glass-strong border-t border-border-subtle flex justify-around py-2 z-50">
                {navItems.map(({ to, icon: Icon, label }) => (
                    <NavLink
                        key={to}
                        to={to}
                        end={to === '/'}
                        className={({ isActive }) =>
                            `flex flex-col items-center gap-1 px-4 py-1.5 rounded-lg text-xs transition-colors ${isActive ? 'text-accent-cyan' : 'text-text-muted'
                            }`
                        }
                    >
                        <Icon className="w-5 h-5" />
                        {label}
                    </NavLink>
                ))}
            </nav>
        </div>
    );
}

