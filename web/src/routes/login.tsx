import { useState, useEffect } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { cn } from '@/lib/utils';
import {
  useAuthCheck,
  useLogin,
  useSetup,
  usePasskeyLogin,
} from '@/hooks';
import { Spinner } from '@/components/ui/spinner';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

export const Route = createFileRoute('/login')({
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const { data: authData, isLoading: isAuthLoading } = useAuthCheck();

  // Redirect to dashboard if already authenticated
  useEffect(() => {
    if (!isAuthLoading && authData?.isAuthenticated) {
      navigate({ to: '/' });
    }
  }, [authData, isAuthLoading, navigate]);

  // Show loading while checking auth
  if (isAuthLoading) {
    return (
      <div className="min-h-screen bg-terminal-bg flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  // Show setup screen if needs setup
  if (authData?.needsSetup) {
    return <SetupScreen />;
  }

  // Show login screen
  return <LoginScreen />;
}

function SetupScreen() {
  const navigate = useNavigate();
  const setup = useSetup();

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!username.trim() || !password.trim()) {
      setError('Username and password are required');
      return;
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    try {
      await setup.mutateAsync({ username, password });
      navigate({ to: '/' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Setup failed');
    }
  };

  return (
    <div className="min-h-screen bg-terminal-bg flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="bg-terminal-surface border border-terminal-border rounded-sm p-8">
          {/* Header */}
          <div className="text-center mb-8">
            <div className="flex items-center justify-center gap-2 mb-4">
              <h1 className="text-xl font-bold text-terminal-text">
                check
              </h1>
            </div>
            <p className="text-terminal-muted text-sm">
              Welcome! Create your admin account to get started.
            </p>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs uppercase tracking-wider text-terminal-muted mb-2">
                Username
              </label>
              <Input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                autoComplete="username"
                autoFocus
              />
            </div>

            <div>
              <label className="block text-xs uppercase tracking-wider text-terminal-muted mb-2">
                Password
              </label>
              <Input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                autoComplete="new-password"
              />
              <p className="text-xs text-terminal-muted mt-1">
                Minimum 8 characters
              </p>
            </div>

            {error && (
              <div className="text-terminal-red text-sm bg-terminal-red/10 border border-terminal-red/20 rounded px-3 py-2">
                {error}
              </div>
            )}

            <Button
              type="submit"
              className="w-full"
              disabled={setup.isPending}
            >
              {setup.isPending ? (
                <span className="flex items-center justify-center gap-2">
                  <Spinner size="sm" />
                  Creating Account...
                </span>
              ) : (
                'Create Account'
              )}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}

function LoginScreen() {
  const navigate = useNavigate();
  const login = useLogin();
  const passkeyLogin = usePasskeyLogin();

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!username.trim() || !password.trim()) {
      setError('Username and password are required');
      return;
    }

    try {
      await login.mutateAsync({ username, password });
      navigate({ to: '/' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    }
  };

  const handlePasskeyLogin = async () => {
    setError('');
    try {
      await passkeyLogin.mutateAsync({ username: '' });
      navigate({ to: '/' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Passkey login failed');
    }
  };

  return (
    <div className="min-h-screen bg-terminal-bg flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="bg-terminal-surface border border-terminal-border rounded-sm p-8">
          {/* Header */}
          <div className="text-center mb-8">
            <div className="flex items-center justify-center gap-2 mb-4">
              <h1 className="text-xl font-bold text-terminal-text">
                check
              </h1>
            </div>
            <p className="text-terminal-muted text-sm">
              Sign in to your account
            </p>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs uppercase tracking-wider text-terminal-muted mb-2">
                Username
              </label>
              <Input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                autoComplete="username"
                autoFocus
              />
            </div>

            <div>
              <label className="block text-xs uppercase tracking-wider text-terminal-muted mb-2">
                Password
              </label>
              <Input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                autoComplete="current-password"
              />
            </div>

            {error && (
              <div className="text-terminal-red text-sm bg-terminal-red/10 border border-terminal-red/20 rounded px-3 py-2">
                {error}
              </div>
            )}

            <Button
              type="submit"
              className="w-full"
              disabled={login.isPending}
            >
              {login.isPending ? (
                <span className="flex items-center justify-center gap-2">
                  <Spinner size="sm" />
                  Signing in...
                </span>
              ) : (
                'Sign In'
              )}
            </Button>
          </form>

          {/* Divider */}
          <div className="relative my-6">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-terminal-border" />
            </div>
            <div className="relative flex justify-center text-xs">
              <span className="bg-terminal-surface px-2 text-terminal-muted">
                or
              </span>
            </div>
          </div>

          {/* Passkey Login */}
          <Button
            type="button"
            variant="outline"
            className="w-full"
            onClick={handlePasskeyLogin}
            disabled={passkeyLogin.isPending}
          >
            {passkeyLogin.isPending ? (
              <span className="flex items-center justify-center gap-2">
                <Spinner size="sm" />
                Authenticating...
              </span>
            ) : (
              <span className="flex items-center justify-center gap-2">
                <PasskeyIcon className="w-4 h-4" />
                Sign in with Passkey
              </span>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}

function PasskeyIcon({ className }: { className?: string }) {
  return (
    <svg
      className={cn(className)}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="12" cy="8" r="4" />
      <path d="M8 14h8" />
      <path d="M12 14v4" />
      <path d="M9 18h6" />
    </svg>
  );
}
