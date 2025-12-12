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
  useProbes,
} from '@/hooks';
import { useToast } from '@/components/ui/toast';
import type { Settings } from '@/types';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

type SettingsTab = 'general' | 'notifications' | 'tailscale' | 'snapshots' | 'api-keys' | 'passkeys' | 'probes';

export function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  const { data: settings, isLoading } = useSettings();
  const updateSettings = useUpdateSettings();
  const { showToast } = useToast();

  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const [formData, setFormData] = useState<Partial<Settings>>({});

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
    { id: 'snapshots', label: 'Snapshots' },
    { id: 'api-keys', label: 'API Keys' },
    { id: 'passkeys', label: 'Passkeys' },
    { id: 'probes', label: 'Probes' },
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
            {activeTab === 'snapshots' && (
              <SnapshotsTab
                formData={formData}
                updateField={updateField}
                onSave={handleSaveSettings}
                isSaving={updateSettings.isPending}
              />
            )}
            {activeTab === 'api-keys' && <APIKeysTab />}
            {activeTab === 'passkeys' && <PasskeysTab />}
            {activeTab === 'probes' && <ProbesTab />}
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
      <Button
        onClick={onSave}
        disabled={isSaving}
        className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
      >
        {isSaving ? <Spinner size="sm" /> : 'Save'}
      </Button>
    </div>
  );
}

// Snapshots Tab
interface SnapshotsTabProps {
  formData: Partial<Settings>;
  updateField: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onSave: () => void;
  isSaving: boolean;
}

function SnapshotsTab({ onSave, isSaving }: SnapshotsTabProps) {
  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        Configure snapshot settings for visual monitoring
      </div>
      <Button
        onClick={onSave}
        disabled={isSaving}
        className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
      >
        {isSaving ? <Spinner size="sm" /> : 'Save'}
      </Button>
    </div>
  );
}

// Notifications Tab
interface NotificationsTabProps {
  formData: Partial<Settings>;
  updateField: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onSave: () => void;
  isSaving: boolean;
}

function NotificationsTab({ formData, updateField, onSave, isSaving }: NotificationsTabProps) {
  const { showToast } = useToast();
  const testDiscord = useTestDiscordWebhook();
  const testGotify = useTestGotify();

  const handleTestDiscord = async () => {
    try {
      await testDiscord.mutateAsync();
      showToast('Discord webhook test sent', 'success');
    } catch (error: any) {
      showToast(error.message || 'Failed to test Discord webhook', 'error');
    }
  };

  const handleTestGotify = async () => {
    try {
      await testGotify.mutateAsync();
      showToast('Gotify notification test sent', 'success');
    } catch (error: any) {
      showToast(error.message || 'Failed to test Gotify', 'error');
    }
  };

  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        Configure notification channels for check status changes
      </div>

      <div className="space-y-4">
        <div>
          <label className="text-xs text-terminal-muted mb-1 block">Discord Webhook URL</label>
          <Input
            type="text"
            value={formData.discord_webhook_url || ''}
            onChange={(e) => updateField('discord_webhook_url', e.target.value)}
            placeholder="https://discord.com/api/webhooks/..."
            className="bg-terminal-surface border-terminal-border text-terminal-text"
          />
          <div className="flex justify-end mt-2">
            <Button
              onClick={handleTestDiscord}
              disabled={testDiscord.isPending || !formData.discord_webhook_url}
              variant="outline"
              className="text-xs bg-terminal-surface border-terminal-border"
            >
              {testDiscord.isPending ? <Spinner size="sm" /> : 'Test'}
            </Button>
          </div>
        </div>

        <div>
          <label className="text-xs text-terminal-muted mb-1 block">Gotify Server URL</label>
          <Input
            type="text"
            value={formData.gotify_server_url || ''}
            onChange={(e) => updateField('gotify_server_url', e.target.value)}
            placeholder="https://gotify.example.com"
            className="bg-terminal-surface border-terminal-border text-terminal-text"
          />
        </div>

        <div>
          <label className="text-xs text-terminal-muted mb-1 block">Gotify Token</label>
          <Input
            type="text"
            value={formData.gotify_token || ''}
            onChange={(e) => updateField('gotify_token', e.target.value)}
            placeholder="Gotify application token"
            className="bg-terminal-surface border-terminal-border text-terminal-text"
          />
          <div className="flex justify-end mt-2">
            <Button
              onClick={handleTestGotify}
              disabled={testGotify.isPending || !formData.gotify_server_url || !formData.gotify_token}
              variant="outline"
              className="text-xs bg-terminal-surface border-terminal-border"
            >
              {testGotify.isPending ? <Spinner size="sm" /> : 'Test'}
            </Button>
          </div>
        </div>
      </div>

      <Button
        onClick={onSave}
        disabled={isSaving}
        className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
      >
        {isSaving ? <Spinner size="sm" /> : 'Save'}
      </Button>
    </div>
  );
}

// Tailscale Tab
interface TailscaleTabProps {
  formData: Partial<Settings>;
  updateField: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onSave: () => void;
  isSaving: boolean;
}

function TailscaleTab({ formData, updateField, onSave, isSaving }: TailscaleTabProps) {
  const { showToast } = useToast();
  const testTailscale = useTestTailscale();

  const handleTest = async () => {
    try {
      await testTailscale.mutateAsync();
      showToast('Tailscale connection test successful', 'success');
    } catch (error: any) {
      showToast(error.message || 'Failed to test Tailscale connection', 'error');
    }
  };

  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        Configure Tailscale integration for device and service monitoring
      </div>

      <div className="space-y-4">
        <div>
          <label className="text-xs text-terminal-muted mb-1 block">Tailscale API Key</label>
          <Input
            type="text"
            value={formData.tailscale_api_key || ''}
            onChange={(e) => updateField('tailscale_api_key', e.target.value)}
            placeholder="tskey-api-..."
            className="bg-terminal-surface border-terminal-border text-terminal-text"
          />
        </div>

        <div>
          <label className="text-xs text-terminal-muted mb-1 block">Tailscale Tailnet</label>
          <Input
            type="text"
            value={formData.tailscale_tailnet || ''}
            onChange={(e) => updateField('tailscale_tailnet', e.target.value)}
            placeholder="your-tailnet.tailscale.com"
            className="bg-terminal-surface border-terminal-border text-terminal-text"
          />
        </div>

        <div className="flex justify-end">
          <Button
            onClick={handleTest}
            disabled={testTailscale.isPending || !formData.tailscale_api_key || !formData.tailscale_tailnet}
            variant="outline"
            className="text-xs bg-terminal-surface border-terminal-border"
          >
            {testTailscale.isPending ? <Spinner size="sm" /> : 'Test Connection'}
          </Button>
        </div>
      </div>

      <Button
        onClick={onSave}
        disabled={isSaving}
        className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
      >
        {isSaving ? <Spinner size="sm" /> : 'Save'}
      </Button>
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
  const [newKey, setNewKey] = useState<string | null>(null);

  const handleCreate = async () => {
    try {
      const result = await createAPIKey.mutateAsync(newKeyName.trim() || 'Default Key');
      setNewKey(result.key);
      setNewKeyName('');
      showToast('API key created', 'success');
    } catch (error: any) {
      showToast(error.message || 'Failed to create API key', 'error');
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
    navigator.clipboard.writeText(newKey!);
    showToast('API key copied to clipboard', 'success');
  };

  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        API keys allow programmatic access to GoCheck
      </div>

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
                    Created: {new Date(key.created_at).toLocaleDateString()}
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
                    Registered: {new Date(passkey.created_at).toLocaleDateString()}
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

// Probes Tab
function ProbesTab() {
  const { probes, isLoading, createProbe, deleteProbe, regenerateToken, isCreating, isDeleting, isRegenerating } = useProbes();
  const { showToast } = useToast();

  const [newRegionCode, setNewRegionCode] = useState('');
  const [newIPAddress, setNewIPAddress] = useState('');
  const [newToken, setNewToken] = useState<string | null>(null);
  const [regeneratedToken, setRegeneratedToken] = useState<{ id: number; token: string } | null>(null);

  const handleCreate = async () => {
    try {
      const result = await createProbe({
        region_code: newRegionCode.trim(),
        ip_address: newIPAddress.trim() || undefined,
      });
      setNewToken(result.token);
      setNewRegionCode('');
      setNewIPAddress('');
      showToast('Probe created successfully', 'success');
    } catch (error: any) {
      showToast(error.message || 'Failed to create probe', 'error');
    }
  };

  const handleDelete = async (id: number, regionCode: string) => {
    if (!confirm(`Delete probe "${regionCode}"?`)) return;
    try {
      await deleteProbe(id);
      showToast('Probe deleted', 'success');
    } catch {
      showToast('Failed to delete probe', 'error');
    }
  };

  const handleRegenerateToken = async (id: number) => {
    try {
      const result = await regenerateToken(id);
      setRegeneratedToken({ id, token: result.token });
      showToast('Token regenerated', 'success');
    } catch {
      showToast('Failed to regenerate token', 'error');
    }
  };

  const copyToken = (token: string) => {
    navigator.clipboard.writeText(token);
    showToast('Token copied to clipboard', 'success');
  };

  return (
    <div className="space-y-6">
      <div className="text-xs text-terminal-muted mb-4">
        Manage multi-region probes. Create probes and generate authentication tokens for deployment.
      </div>

      <div className="space-y-3">
        <Input
          type="text"
          value={newRegionCode}
          onChange={(e) => setNewRegionCode(e.target.value)}
          placeholder="Region code (e.g., us-nyc-1)"
          className="bg-terminal-surface border-terminal-border text-terminal-text"
        />
        <Input
          type="text"
          value={newIPAddress}
          onChange={(e) => setNewIPAddress(e.target.value)}
          placeholder="IP address (optional)"
          className="bg-terminal-surface border-terminal-border text-terminal-text"
        />
        <Button
          onClick={handleCreate}
          disabled={isCreating || !newRegionCode.trim()}
          className="bg-terminal-green text-terminal-bg font-bold hover:opacity-90 w-full"
        >
          {isCreating ? <Spinner size="sm" /> : 'Create Probe'}
        </Button>
      </div>

      {newToken && (
        <div className="p-4 bg-terminal-yellow/10 border border-terminal-yellow rounded">
          <div className="text-xs text-terminal-yellow mb-2">
            Save this token - it won't be shown again!
          </div>
          <div className="flex gap-2">
            <code className="flex-1 bg-terminal-bg p-2 rounded text-xs font-mono text-terminal-text break-all">
              {newToken}
            </code>
            <Button
              onClick={() => {
                copyToken(newToken);
                setNewToken(null);
              }}
              variant="outline"
              className="text-xs bg-terminal-surface border-terminal-border"
            >
              Copy
            </Button>
          </div>
        </div>
      )}

      {regeneratedToken && (
        <div className="p-4 bg-terminal-yellow/10 border border-terminal-yellow rounded">
          <div className="text-xs text-terminal-yellow mb-2">
            New token for probe (save this - it won't be shown again!)
          </div>
          <div className="flex gap-2">
            <code className="flex-1 bg-terminal-bg p-2 rounded text-xs font-mono text-terminal-text break-all">
              {regeneratedToken.token}
            </code>
            <Button
              onClick={() => {
                copyToken(regeneratedToken.token);
                setRegeneratedToken(null);
              }}
              variant="outline"
              className="text-xs bg-terminal-surface border-terminal-border"
            >
              Copy
            </Button>
          </div>
        </div>
      )}

      <div>
        <div className="text-xs text-terminal-muted mb-3">Registered Probes</div>
        {isLoading ? (
          <Spinner />
        ) : probes.length === 0 ? (
          <div className="text-sm text-terminal-muted">No probes registered</div>
        ) : (
          <div className="space-y-2">
            {probes.map((probe) => (
              <div
                key={probe.id}
                className="p-3 bg-terminal-surface border border-terminal-border rounded"
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <div className="text-sm font-semibold">{probe.region_code}</div>
                      <div
                        className={cn(
                          'text-[10px] px-2 py-0.5 rounded',
                          probe.status === 'ONLINE'
                            ? 'bg-terminal-green/20 text-terminal-green'
                            : 'bg-terminal-red/20 text-terminal-red'
                        )}
                      >
                        {probe.status}
                      </div>
                    </div>
                    {probe.ip_address && (
                      <div className="text-[10px] text-terminal-muted mt-1">
                        IP: {probe.ip_address}
                      </div>
                    )}
                    {probe.last_seen_at && (
                      <div className="text-[10px] text-terminal-muted mt-1">
                        Last seen: {new Date(probe.last_seen_at).toLocaleString()}
                      </div>
                    )}
                  </div>
                  <div className="flex gap-2">
                    <Button
                      onClick={() => handleRegenerateToken(probe.id)}
                      disabled={isRegenerating}
                      variant="outline"
                      className="text-xs bg-terminal-surface border-terminal-border"
                    >
                      {isRegenerating ? <Spinner size="sm" /> : 'Regenerate Token'}
                    </Button>
                    <Button
                      onClick={() => handleDelete(probe.id, probe.region_code)}
                      disabled={isDeleting}
                      variant="outline"
                      className="text-xs bg-terminal-red/10 border-terminal-red/50 text-terminal-red hover:bg-terminal-red/20"
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
