import { useState } from 'react';
import { Modal, ModalHeader } from '@/components/ui/modal';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { cn } from '@/lib/utils';
import {
  useSettings,
  useUpdateSettings,
  useAPIKeys,
  useCreateAPIKey,
  useDeleteAPIKey,
  usePasskeys,
  useDeletePasskey,
  useRegisterPasskey,
  useTestDiscordWebhook,
  useTestGotify,
  useTestTailscale,
} from '@/hooks';
import { useToast } from '@/components/ui/toast';
import type { Settings } from '@/types';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

type SettingsTab = 'general' | 'notifications' | 'tailscale' | 'api-keys' | 'passkeys';

export function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  const { data: settings, isLoading } = useSettings();
  const updateSettings = useUpdateSettings();
  const { showToast } = useToast();

  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const [formData, setFormData] = useState<Partial<Settings>>({});

  // Initialize form data when settings load
  useState(() => {
    if (settings) {
      setFormData(settings);
    }
  });

  const handleSaveSettings = async () => {
    try {
      await updateSettings.mutateAsync(formData as Settings);
      showToast('Settings saved', 'success');
    } catch (error) {
      showToast('Failed to save settings', 'error');
    }
  };

  const updateField = <K extends keyof Settings>(key: K, value: Settings[K]) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  const tabs: { id: SettingsTab; label: string }[] = [
    { id: 'general', label: 'General' },
    { id: 'notifications', label: 'Notifications' },
    { id: 'tailscale', label: 'Tailscale' },
    { id: 'api-keys', label: 'API Keys' },
    { id: 'passkeys', label: 'Passkeys' },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg">
      <ModalHeader onClose={onClose}>$ settings</ModalHeader>

      {isLoading ? (
        <div className="p-6 flex justify-center">
          <Spinner size="lg" />
        </div>
      ) : (
        <>
          {/* Tab Navigation */}
          <div className="border-b border-terminal-border">
            <div className="flex overflow-x-auto">
              {tabs.map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={cn(
                    'px-4 py-3 text-xs uppercase tracking-widest transition whitespace-nowrap',
                    activeTab === tab.id
                      ? 'text-terminal-green border-b-2 border-terminal-green'
                      : 'text-terminal-muted hover:text-terminal-text'
                  )}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>

          <div className="p-6 max-h-[60vh] overflow-y-auto">
            {activeTab === 'general' && (
              <GeneralTab
                formData={formData}
                updateField={updateField}
                onSave={handleSaveSettings}
                isSaving={updateSettings.isPending}
              />
            )}
            {activeTab === 'notifications' && (
              <NotificationsTab
                formData={formData}
                updateField={updateField}
                onSave={handleSaveSettings}
                isSaving={updateSettings.isPending}
              />
            )}
            {activeTab === 'tailscale' && (
              <TailscaleTab
                formData={formData}
                updateField={updateField}
                onSave={handleSaveSettings}
                isSaving={updateSettings.isPending}
              />
            )}
            {activeTab === 'api-keys' && <APIKeysTab />}
            {activeTab === 'passkeys' && <PasskeysTab />}
          </div>
        </>
      )}
    </Modal>
  );
}

// General Tab
interface GeneralTabProps {
  formData: Partial<Settings>;
  updateField: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onSave: () => void;
  isSaving: boolean;
}

function GeneralTab({ onSave, isSaving }: GeneralTabProps) {
  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        General application settings are configured in the backend config.yaml
      </div>

      <div className="flex justify-end pt-4 border-t border-terminal-border">
        <Button
          onClick={onSave}
          disabled={isSaving}
          className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
        >
          {isSaving ? <Spinner size="sm" /> : 'Save Settings'}
        </Button>
      </div>
    </div>
  );
}

// Notifications Tab
interface NotificationsTabProps {
  formData: Partial<Settings>;
  updateField: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onSave: () => Promise<void>;
  isSaving: boolean;
}

function NotificationsTab({ formData, updateField, onSave, isSaving }: NotificationsTabProps) {
  const testDiscord = useTestDiscordWebhook();
  const testGotify = useTestGotify();
  const { showToast } = useToast();

  const handleTestDiscord = async () => {
    if (!formData.discord_webhook_url) {
      showToast('Enter a Discord webhook URL first', 'error');
      return;
    }
    try {
      // Save settings first, then test (test uses saved settings)
      await onSave();
      await testDiscord.mutateAsync();
      showToast('Discord test message sent', 'success');
    } catch {
      showToast('Failed to send test message', 'error');
    }
  };

  const handleTestGotify = async () => {
    if (!formData.gotify_url || !formData.gotify_token) {
      showToast('Enter Gotify URL and token first', 'error');
      return;
    }
    try {
      // Save settings first, then test (test uses saved settings)
      await onSave();
      await testGotify.mutateAsync();
      showToast('Gotify test message sent', 'success');
    } catch {
      showToast('Failed to send test message', 'error');
    }
  };

  return (
    <div className="space-y-6">
      {/* Discord */}
      <div>
        <div className="text-xs text-terminal-purple uppercase tracking-widest mb-4">
          Discord Webhook
        </div>
        <div className="space-y-3">
          <div>
            <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
              Webhook URL
            </label>
            <Input
              type="url"
              value={formData.discord_webhook_url || ''}
              onChange={(e) => updateField('discord_webhook_url', e.target.value)}
              className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
              placeholder="https://discord.com/api/webhooks/..."
            />
          </div>
          <Button
            type="button"
            onClick={handleTestDiscord}
            disabled={testDiscord.isPending || !formData.discord_webhook_url}
            variant="outline"
            className="text-xs bg-terminal-surface border-terminal-border text-terminal-muted hover:text-terminal-text"
          >
            {testDiscord.isPending ? <Spinner size="sm" /> : 'Test Discord'}
          </Button>
        </div>
      </div>

      {/* Gotify */}
      <div>
        <div className="text-xs text-terminal-cyan uppercase tracking-widest mb-4">
          Gotify
        </div>
        <div className="space-y-3">
          <div>
            <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
              Server URL
            </label>
            <Input
              type="url"
              value={formData.gotify_url || ''}
              onChange={(e) => updateField('gotify_url', e.target.value)}
              className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
              placeholder="https://gotify.example.com"
            />
          </div>
          <div>
            <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
              App Token
            </label>
            <Input
              type="password"
              value={formData.gotify_token || ''}
              onChange={(e) => updateField('gotify_token', e.target.value)}
              className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
              placeholder="••••••••"
            />
          </div>
          <Button
            type="button"
            onClick={handleTestGotify}
            disabled={
              testGotify.isPending ||
              !formData.gotify_url ||
              !formData.gotify_token
            }
            variant="outline"
            className="text-xs bg-terminal-surface border-terminal-border text-terminal-muted hover:text-terminal-text"
          >
            {testGotify.isPending ? <Spinner size="sm" /> : 'Test Gotify'}
          </Button>
        </div>
      </div>

      <div className="flex justify-end pt-4 border-t border-terminal-border">
        <Button
          onClick={onSave}
          disabled={isSaving}
          className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
        >
          {isSaving ? <Spinner size="sm" /> : 'Save Settings'}
        </Button>
      </div>
    </div>
  );
}

// Tailscale Tab
interface TailscaleTabProps {
  formData: Partial<Settings>;
  updateField: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onSave: () => Promise<void>;
  isSaving: boolean;
}

function TailscaleTab({ formData, updateField, onSave, isSaving }: TailscaleTabProps) {
  const testTailscale = useTestTailscale();
  const { showToast } = useToast();

  const handleTest = async () => {
    if (!formData.tailscale_api_key || !formData.tailscale_tailnet) {
      showToast('Enter Tailscale API key and tailnet first', 'error');
      return;
    }
    try {
      // Save settings first, then test (test uses saved settings)
      await onSave();
      await testTailscale.mutateAsync();
      showToast('Tailscale connection successful', 'success');
    } catch {
      showToast('Failed to connect to Tailscale', 'error');
    }
  };

  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        Configure Tailscale API for device status monitoring
      </div>

      <div>
        <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
          API Key
        </label>
        <Input
          type="password"
          value={formData.tailscale_api_key || ''}
          onChange={(e) => updateField('tailscale_api_key', e.target.value)}
          className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
          placeholder="tskey-api-..."
        />
        <p className="text-[10px] text-terminal-muted mt-1">
          Generate at{' '}
          <a
            href="https://login.tailscale.com/admin/settings/keys"
            target="_blank"
            rel="noopener noreferrer"
            className="text-terminal-cyan hover:underline"
          >
            admin.tailscale.com
          </a>
        </p>
      </div>

      <div>
        <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
          Tailnet Name
        </label>
        <Input
          type="text"
          value={formData.tailscale_tailnet || ''}
          onChange={(e) => updateField('tailscale_tailnet', e.target.value)}
          className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
          placeholder="example.com or yourname@github"
        />
      </div>

      <Button
        type="button"
        onClick={handleTest}
        disabled={
          testTailscale.isPending ||
          !formData.tailscale_api_key ||
          !formData.tailscale_tailnet
        }
        variant="outline"
        className="text-xs bg-terminal-surface border-terminal-border text-terminal-muted hover:text-terminal-text"
      >
        {testTailscale.isPending ? <Spinner size="sm" /> : 'Test Connection'}
      </Button>

      <div className="flex justify-end pt-4 border-t border-terminal-border">
        <Button
          onClick={onSave}
          disabled={isSaving}
          className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
        >
          {isSaving ? <Spinner size="sm" /> : 'Save Settings'}
        </Button>
      </div>
    </div>
  );
}

// API Keys Tab
function APIKeysTab() {
  const { data: apiKeys = [], isLoading } = useAPIKeys();
  const createAPIKey = useCreateAPIKey();
  const deleteAPIKey = useDeleteAPIKey();
  const { showToast } = useToast();

  const [newKeyName, setNewKeyName] = useState('');
  const [newKey, setNewKey] = useState('');

  const handleCreate = async () => {
    if (!newKeyName.trim()) {
      showToast('Enter a name for the API key', 'error');
      return;
    }
    try {
      const result = await createAPIKey.mutateAsync(newKeyName.trim());
      setNewKey(result.key);
      setNewKeyName('');
      showToast('API key created', 'success');
    } catch {
      showToast('Failed to create API key', 'error');
    }
  };

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Delete API key "${name}"?`)) return;
    try {
      await deleteAPIKey.mutateAsync(id);
      showToast('API key deleted', 'success');
    } catch {
      showToast('Failed to delete API key', 'error');
    }
  };

  const copyKey = () => {
    navigator.clipboard.writeText(newKey);
    showToast('API key copied to clipboard', 'success');
  };

  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        API keys allow programmatic access to GoCheck
      </div>

      {/* Create new key */}
      <div className="flex gap-3">
        <Input
          type="text"
          value={newKeyName}
          onChange={(e) => setNewKeyName(e.target.value)}
          placeholder="API key name"
          className="flex-1 bg-terminal-surface border-terminal-border text-terminal-text"
        />
        <Button
          onClick={handleCreate}
          disabled={createAPIKey.isPending || !newKeyName.trim()}
          className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
        >
          {createAPIKey.isPending ? <Spinner size="sm" /> : 'Create'}
        </Button>
      </div>

      {/* New key display */}
      {newKey && (
        <div className="p-4 bg-terminal-yellow/10 border border-terminal-yellow rounded">
          <div className="text-xs text-terminal-yellow mb-2">
            Save this key - it won't be shown again!
          </div>
          <div className="flex gap-2">
            <code className="flex-1 bg-terminal-bg p-2 rounded text-xs font-mono text-terminal-text break-all">
              {newKey}
            </code>
            <Button
              onClick={copyKey}
              variant="outline"
              className="text-xs bg-terminal-surface border-terminal-border"
            >
              Copy
            </Button>
          </div>
        </div>
      )}

      {/* Existing keys */}
      <div>
        <div className="text-xs text-terminal-muted mb-3">Existing Keys</div>
        {isLoading ? (
          <Spinner />
        ) : apiKeys.length === 0 ? (
          <div className="text-sm text-terminal-muted">No API keys created</div>
        ) : (
          <div className="space-y-2">
            {apiKeys.map((key) => (
              <div
                key={key.id}
                className="flex items-center justify-between p-3 bg-terminal-surface border border-terminal-border rounded"
              >
                <div>
                  <div className="text-sm font-semibold">{key.name}</div>
                  <div className="text-[10px] text-terminal-muted">
                    Created:{' '}
                    {new Date(key.created_at).toLocaleDateString()}
                  </div>
                </div>
                <Button
                  onClick={() => handleDelete(key.id, key.name)}
                  disabled={deleteAPIKey.isPending}
                  variant="outline"
                  className="text-xs bg-terminal-red/10 border-terminal-red/50 text-terminal-red hover:bg-terminal-red/20"
                >
                  Delete
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// Passkeys Tab
function PasskeysTab() {
  const { data: passkeys = [], isLoading } = usePasskeys();
  const deletePasskey = useDeletePasskey();
  const registerPasskey = useRegisterPasskey();
  const { showToast } = useToast();

  const [newKeyName, setNewKeyName] = useState('');

  const handleRegister = async () => {
    try {
      await registerPasskey.mutateAsync({ name: newKeyName.trim() || 'Default Passkey' });
      setNewKeyName('');
      showToast('Passkey registered', 'success');
    } catch (error: any) {
      showToast(error.message || 'Failed to register passkey', 'error');
    }
  };

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Delete passkey "${name}"?`)) return;
    try {
      await deletePasskey.mutateAsync(id);
      showToast('Passkey deleted', 'success');
    } catch {
      showToast('Failed to delete passkey', 'error');
    }
  };

  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        Passkeys provide passwordless authentication using your device's biometrics or security key
      </div>

      {/* Register new passkey */}
      <div className="flex gap-3">
        <Input
          type="text"
          value={newKeyName}
          onChange={(e) => setNewKeyName(e.target.value)}
          placeholder="Passkey name (optional)"
          className="flex-1 bg-terminal-surface border-terminal-border text-terminal-text"
        />
        <Button
          onClick={handleRegister}
          disabled={registerPasskey.isPending}
          className="bg-terminal-cyan text-terminal-bg font-bold hover:opacity-90"
        >
          {registerPasskey.isPending ? <Spinner size="sm" /> : 'Register'}
        </Button>
      </div>

      {/* Existing passkeys */}
      <div>
        <div className="text-xs text-terminal-muted mb-3">Registered Passkeys</div>
        {isLoading ? (
          <Spinner />
        ) : passkeys.length === 0 ? (
          <div className="text-sm text-terminal-muted">No passkeys registered</div>
        ) : (
          <div className="space-y-2">
            {passkeys.map((passkey) => (
              <div
                key={passkey.id}
                className="flex items-center justify-between p-3 bg-terminal-surface border border-terminal-border rounded"
              >
                <div>
                  <div className="text-sm font-semibold">{passkey.name}</div>
                  <div className="text-[10px] text-terminal-muted">
                    Registered:{' '}
                    {new Date(passkey.created_at).toLocaleDateString()}
                    {passkey.last_used_at && (
                      <>
                        {' • Last used: '}
                        {new Date(passkey.last_used_at).toLocaleDateString()}
                      </>
                    )}
                  </div>
                </div>
                <Button
                  onClick={() => handleDelete(passkey.id, passkey.name)}
                  disabled={deletePasskey.isPending}
                  variant="outline"
                  className="text-xs bg-terminal-red/10 border-terminal-red/50 text-terminal-red hover:bg-terminal-red/20"
                >
                  Delete
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
