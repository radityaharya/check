import { useEffect } from 'react';
import { Modal } from '@/components/ui/modal';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { cn } from '@/lib/utils';
import { parseStatusCodes } from '@/lib/helpers';
import { useGroups, useTags, useTailscaleDevices, useSettings } from '@/hooks';
import { useAppForm } from '@/hooks/form';
import type { Check, CheckFormData, CheckType } from '@/types';

interface MonitorFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (data: CheckFormData) => Promise<void>;
  editingCheck?: Check | null;
}

const defaultFormData: CheckFormData = {
  name: '',
  type: 'http',
  url: '',
  interval_seconds: 60,
  timeout_seconds: 10,
  retries: 0,
  retry_delay_seconds: 5,
  enabled: true,
  method: 'GET',
  expected_status_codes: [200],
  json_path: '',
  expected_json_value: '',
  host: '',
  postgres_conn_string: '',
  postgres_query: '',
  expected_query_value: '',
  dns_hostname: '',
  dns_record_type: 'A',
  expected_dns_value: '',
  group_id: '',
  tag_ids: [],
  tailscale_device_id: '',
  tailscale_service_host: '',
  tailscale_service_port: 80,
  tailscale_service_protocol: 'http',
  tailscale_service_path: '/',
};

export function MonitorFormModal({
  isOpen,
  onClose,
  onSave,
  editingCheck,
}: MonitorFormModalProps) {
  const { data: groups = [] } = useGroups();
  const { data: tags = [] } = useTags();
  const { data: settings } = useSettings();
  const hasTailscale =
    !!settings?.tailscale_api_key && !!settings?.tailscale_tailnet;
  const {
    data: tailscaleDevices = [],
    isLoading: loadingDevices,
    refetch: refetchDevices,
  } = useTailscaleDevices(hasTailscale);

  const form = useAppForm({
    defaultValues: defaultFormData,
    onSubmit: async ({ value }) => {
      await onSave(value);
      onClose();
    },
  });

  // Reset form when modal opens/closes or editingCheck changes
  useEffect(() => {
    if (isOpen) {
      if (editingCheck) {
        form.reset({
          name: editingCheck.name,
          type: editingCheck.type || 'http',
          url: editingCheck.url || '',
          interval_seconds: editingCheck.interval_seconds,
          timeout_seconds: editingCheck.timeout_seconds,
          retries: editingCheck.retries ?? 0,
          retry_delay_seconds: editingCheck.retry_delay_seconds ?? 5,
          enabled: editingCheck.enabled,
          method: editingCheck.method || 'GET',
          expected_status_codes: editingCheck.expected_status_codes || [200],
          json_path: editingCheck.json_path || '',
          expected_json_value: editingCheck.expected_json_value || '',
          host: editingCheck.host || '',
          postgres_conn_string: editingCheck.postgres_conn_string || '',
          postgres_query: editingCheck.postgres_query || '',
          expected_query_value: editingCheck.expected_query_value || '',
          dns_hostname: editingCheck.dns_hostname || '',
          dns_record_type: editingCheck.dns_record_type || 'A',
          expected_dns_value: editingCheck.expected_dns_value || '',
          group_id: editingCheck.group_id || '',
          tag_ids: (editingCheck.tags || []).map((t) => t.id),
          tailscale_device_id: editingCheck.tailscale_device_id || '',
          tailscale_service_host: editingCheck.tailscale_service_host || '',
          tailscale_service_port: editingCheck.tailscale_service_port || 80,
          tailscale_service_protocol:
            editingCheck.tailscale_service_protocol || 'http',
          tailscale_service_path: editingCheck.tailscale_service_path || '/',
        });
      } else {
        form.reset(defaultFormData);
      }
    }
  }, [isOpen, editingCheck]);

  const checkTypes: {
    type: CheckType;
    label: string;
    color: string;
    description: string;
  }[] = [
    {
      type: 'http',
      label: 'HTTP(S)',
      color: 'terminal-blue',
      description: 'Web endpoints',
    },
    {
      type: 'ping',
      label: 'Ping',
      color: 'terminal-cyan',
      description: 'ICMP ping',
    },
    {
      type: 'postgres',
      label: 'PostgreSQL',
      color: 'terminal-purple',
      description: 'Database',
    },
    {
      type: 'json_http',
      label: 'JSON API',
      color: 'terminal-yellow',
      description: 'JSON assertion',
    },
    {
      type: 'dns',
      label: 'DNS',
      color: 'terminal-green',
      description: 'DNS resolver',
    },
    {
      type: 'tailscale',
      label: 'Tailscale',
      color: 'terminal-cyan',
      description: 'Device status',
    },
    {
      type: 'tailscale_service',
      label: 'TS Service',
      color: 'terminal-purple',
      description: 'Tailnet service',
    },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} fullScreen>
      <div className="min-h-screen">
        <div className="bg-terminal-surface border-b border-terminal-border sticky top-0 z-10">
          <div className="container mx-auto px-6 py-4 flex justify-between items-center">
            <h2 className="text-lg font-bold text-terminal-green">
              {editingCheck ? 'edit monitor' : 'new monitor'}
            </h2>
            <button
              onClick={onClose}
              className="text-terminal-muted hover:text-terminal-text text-2xl"
            >
              ×
            </button>
          </div>
        </div>

        <div className="container mx-auto px-6 py-8 max-w-4xl">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              form.handleSubmit();
            }}
          >
            {/* Monitor Type Selection */}
            <div className="mb-10">
              <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-4">
                Monitor Type
              </label>
              <form.Field name="type">
                {(field) => (
                  <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                    {checkTypes.map((ct) => (
                      <button
                        key={ct.type}
                        type="button"
                        onClick={() => field.handleChange(ct.type)}
                        className={cn(
                          'p-4 rounded-sm border-2 transition text-left',
                          field.state.value === ct.type
                            ? `border-${ct.color} bg-${ct.color}/10`
                            : 'border-terminal-border hover:border-terminal-muted'
                        )}
                      >
                        <div className={`text-${ct.color} font-bold mb-1`}>
                          {ct.label}
                        </div>
                        <div className="text-[10px] text-terminal-muted">
                          {ct.description}
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </form.Field>
            </div>

            {/* Basic Settings */}
            <div className="grid md:grid-cols-2 gap-6 mb-8">
              <form.AppField
                name="name"
                children={(field) => (
                  <field.TextField
                    label="Display Name"
                    placeholder="My Server"
                    required
                  />
                )}
              />
              <div className="grid grid-cols-4 gap-4">
                <form.AppField
                  name="interval_seconds"
                  children={(field) => (
                    <field.NumberField label="Interval (s)" min={10} required />
                  )}
                />
                <form.AppField
                  name="timeout_seconds"
                  children={(field) => (
                    <field.NumberField label="Timeout (s)" min={1} required />
                  )}
                />
                <form.AppField
                  name="retries"
                  children={(field) => (
                    <field.NumberField label="Retries" min={0} max={10} />
                  )}
                />
                <form.AppField
                  name="retry_delay_seconds"
                  children={(field) => (
                    <field.NumberField label="Retry Delay" min={1} max={60} />
                  )}
                />
              </div>
            </div>

            {/* Group & Tags */}
            <div className="grid md:grid-cols-2 gap-6 mb-8">
              <form.AppField
                name="group_id"
                children={(field) => (
                  <field.SelectField
                    label="Group"
                    placeholder="No Group"
                    options={groups.map((g) => ({
                      value: g.id.toString(),
                      label: g.name,
                    }))}
                  />
                )}
              />
              <form.AppField
                name="tag_ids"
                children={(field) => (
                  <field.TagsField label="Tags" tags={tags} />
                )}
              />
            </div>

            {/* Type-specific Fields */}
            <form.Subscribe selector={(state) => state.values.type}>
              {(type) => (
                <>
                  {/* HTTP / JSON HTTP Fields */}
                  {(type === 'http' || type === 'json_http') && (
                    <HTTPFields
                      form={form}
                      isJsonHttp={type === 'json_http'}
                    />
                  )}

                  {/* Ping Fields */}
                  {type === 'ping' && <PingFields form={form} />}

                  {/* DNS Fields */}
                  {type === 'dns' && <DNSFields form={form} />}

                  {/* PostgreSQL Fields */}
                  {type === 'postgres' && <PostgresFields form={form} />}

                  {/* Tailscale Fields */}
                  {type === 'tailscale' && (
                    <TailscaleFields
                      form={form}
                      tailscaleDevices={tailscaleDevices}
                      loadingDevices={loadingDevices}
                      refetchDevices={refetchDevices}
                      hasTailscale={hasTailscale}
                    />
                  )}

                  {/* Tailscale Service Fields */}
                  {type === 'tailscale_service' && (
                    <TailscaleServiceFields
                      form={form}
                      tailscaleDevices={tailscaleDevices}
                      loadingDevices={loadingDevices}
                      refetchDevices={refetchDevices}
                      hasTailscale={hasTailscale}
                    />
                  )}
                </>
              )}
            </form.Subscribe>

            {/* Active Monitoring Toggle */}
            <div className="p-4 bg-terminal-surface border border-terminal-border rounded-sm mb-8">
              <form.AppField
                name="enabled"
                children={(field) => (
                  <field.CheckboxField label="Active Monitoring" />
                )}
              />
            </div>

            {/* Submit Buttons */}
            <div className="flex gap-4">
              <form.Subscribe selector={(state) => state.isSubmitting}>
                {(isSubmitting) => (
                  <Button
                    type="submit"
                    disabled={isSubmitting}
                    className="flex-1 bg-terminal-green text-terminal-bg font-bold hover:opacity-90 text-sm uppercase tracking-wide py-4"
                  >
                    {isSubmitting ? 'Saving...' : 'Save Monitor'}
                  </Button>
                )}
              </form.Subscribe>
              <Button
                type="button"
                onClick={onClose}
                variant="outline"
                className="px-6 py-4 bg-terminal-surface border-terminal-border text-terminal-muted hover:text-terminal-text text-sm uppercase tracking-wide"
              >
                Cancel
              </Button>
            </div>
          </form>
        </div>
      </div>
    </Modal>
  );
}

// Type-specific field components
// biome-ignore lint/suspicious/noExplicitAny: Form typing is complex
function HTTPFields({ form, isJsonHttp }: { form: any; isJsonHttp: boolean }) {
  return (
    <div className="space-y-6 mb-8 p-6 bg-terminal-surface border border-terminal-border rounded-sm">
      <div className="text-xs text-terminal-muted uppercase tracking-widest mb-4">
        HTTP Configuration
      </div>
      <div className="grid md:grid-cols-4 gap-4">
        <div className="md:col-span-3">
          <form.AppField
            name="url"
            children={(field: any) => (
              <field.TextField
                label="URL"
                type="url"
                placeholder="https://api.example.com/health"
                required
              />
            )}
          />
        </div>
        <form.AppField
          name="method"
          children={(field: any) => (
            <field.SelectField
              label="Method"
              options={[
                { value: 'GET', label: 'GET' },
                { value: 'POST', label: 'POST' },
                { value: 'PUT', label: 'PUT' },
                { value: 'HEAD', label: 'HEAD' },
              ]}
            />
          )}
        />
      </div>

      {!isJsonHttp && (
        <form.Field name="expected_status_codes">
          {(field: any) => (
            <div>
              <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
                Expected Status Codes
              </label>
              <input
                type="text"
                value={(field.state.value || [200]).join(', ')}
                onChange={(e) =>
                  field.handleChange(parseStatusCodes(e.target.value))
                }
                className="w-full bg-terminal-bg border border-terminal-border text-terminal-text px-4 py-2 rounded focus:border-terminal-green outline-none transition font-mono"
                placeholder="200, 201, 204"
              />
              <div className="text-[10px] text-terminal-muted mt-1">
                Comma-separated list
              </div>
            </div>
          )}
        </form.Field>
      )}

      {isJsonHttp && (
        <div className="space-y-4 pt-4 border-t border-terminal-border">
          <div className="text-xs text-terminal-yellow uppercase tracking-widest">
            JSON Assertion
          </div>
          <div className="grid md:grid-cols-2 gap-4">
            <form.AppField
              name="json_path"
              children={(field: any) => (
                <field.TextField label="JSON Path" placeholder="data.status" />
              )}
            />
            <form.AppField
              name="expected_json_value"
              children={(field: any) => (
                <field.TextField label="Expected Value" placeholder="ok" />
              )}
            />
          </div>
        </div>
      )}
    </div>
  );
}

// biome-ignore lint/suspicious/noExplicitAny: Form typing is complex
function PingFields({ form }: { form: any }) {
  return (
    <div className="space-y-6 mb-8 p-6 bg-terminal-surface border border-terminal-border rounded-sm">
      <div className="text-xs text-terminal-cyan uppercase tracking-widest mb-4">
        Ping Configuration
      </div>
      <form.AppField
        name="host"
        children={(field: any) => (
          <field.TextField
            label="Host / IP Address"
            placeholder="8.8.8.8 or google.com"
            required
          />
        )}
      />
    </div>
  );
}

// biome-ignore lint/suspicious/noExplicitAny: Form typing is complex
function DNSFields({ form }: { form: any }) {
  return (
    <div className="space-y-6 mb-8 p-6 bg-terminal-surface border border-terminal-border rounded-sm">
      <div className="text-xs text-terminal-green uppercase tracking-widest mb-4">
        DNS Configuration
      </div>
      <div className="grid md:grid-cols-2 gap-4">
        <form.AppField
          name="dns_hostname"
          children={(field: any) => (
            <field.TextField
              label="Hostname"
              placeholder="example.com"
              required
            />
          )}
        />
        <form.AppField
          name="dns_record_type"
          children={(field: any) => (
            <field.SelectField
              label="Record Type"
              options={[
                { value: 'A', label: 'A (IPv4)' },
                { value: 'AAAA', label: 'AAAA (IPv6)' },
                { value: 'CNAME', label: 'CNAME' },
                { value: 'MX', label: 'MX' },
                { value: 'TXT', label: 'TXT' },
              ]}
            />
          )}
        />
      </div>
      <form.AppField
        name="expected_dns_value"
        children={(field: any) => (
          <field.TextField
            label="Expected Value (Optional)"
            placeholder="1.2.3.4"
          />
        )}
      />
    </div>
  );
}

// biome-ignore lint/suspicious/noExplicitAny: Form typing is complex
function PostgresFields({ form }: { form: any }) {
  return (
    <div className="space-y-6 mb-8 p-6 bg-terminal-surface border border-terminal-border rounded-sm">
      <div className="text-xs text-terminal-purple uppercase tracking-widest mb-4">
        PostgreSQL Configuration
      </div>
      <form.AppField
        name="postgres_conn_string"
        children={(field: any) => (
          <field.TextField
            label="Connection String"
            placeholder="postgres://user:pass@localhost:5432/dbname"
            required
          />
        )}
      />
      <div className="pt-4 border-t border-terminal-border">
        <div className="text-xs text-terminal-purple uppercase tracking-widest mb-4">
          Query Assertion (Optional)
        </div>
        <div className="space-y-4">
          <form.AppField
            name="postgres_query"
            children={(field: any) => (
              <field.TextAreaField
                label="SQL Query"
                placeholder="SELECT 1"
                rows={2}
              />
            )}
          />
          <form.AppField
            name="expected_query_value"
            children={(field: any) => (
              <field.TextField label="Expected Value" placeholder="1" />
            )}
          />
        </div>
      </div>
    </div>
  );
}

interface TailscaleFieldsProps {
  // biome-ignore lint/suspicious/noExplicitAny: Form typing is complex
  form: any;
  tailscaleDevices: Array<{
    id: string;
    name: string;
    hostname: string;
    addresses: string[];
    online: boolean;
  }>;
  loadingDevices: boolean;
  refetchDevices: () => void;
  hasTailscale: boolean;
}

function TailscaleFields({
  form,
  tailscaleDevices,
  loadingDevices,
  refetchDevices,
  hasTailscale,
}: TailscaleFieldsProps) {
  return (
    <div className="space-y-6 mb-8 p-6 bg-terminal-surface border border-terminal-border rounded-sm">
      <div className="text-xs text-terminal-cyan uppercase tracking-widest mb-4">
        Tailscale Configuration
      </div>
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="block text-[10px] uppercase text-terminal-muted tracking-widest">
            Device
          </label>
          <button
            type="button"
            onClick={() => refetchDevices()}
            disabled={loadingDevices}
            className="text-[10px] text-terminal-cyan hover:text-terminal-green disabled:opacity-50 flex items-center gap-1"
          >
            {loadingDevices && <Spinner size="sm" />}
            <span>{loadingDevices ? 'Loading...' : '↻ Refresh'}</span>
          </button>
        </div>
        {tailscaleDevices.length > 0 ? (
          <form.AppField
            name="tailscale_device_id"
            children={(field: any) => (
              <field.SelectField
                label=""
                placeholder="Select a device..."
                options={tailscaleDevices.map((device) => ({
                  value: device.id,
                  label: `${device.hostname || device.name} (${device.addresses[0] || 'no IP'}) - ${device.online ? '🟢 Online' : '🔴 Offline'}`,
                }))}
                required
              />
            )}
          />
        ) : (
          <form.AppField
            name="tailscale_device_id"
            children={(field: any) => (
              <field.TextField
                label=""
                placeholder="Device ID (configure Tailscale in Settings first)"
                required
              />
            )}
          />
        )}
      </div>
      {!hasTailscale && (
        <div className="p-3 bg-terminal-yellow/10 border border-terminal-yellow/30 rounded text-[10px] text-terminal-yellow">
          Configure Tailscale API credentials in Settings before using this
          check type
        </div>
      )}
    </div>
  );
}

function TailscaleServiceFields({
  form,
  tailscaleDevices,
  loadingDevices,
  refetchDevices,
  hasTailscale,
}: TailscaleFieldsProps) {
  return (
    <div className="space-y-6 mb-8 p-6 bg-terminal-surface border border-terminal-border rounded-sm">
      <div className="text-xs text-terminal-purple uppercase tracking-widest mb-4">
        Tailscale Service Configuration
      </div>
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="block text-[10px] uppercase text-terminal-muted tracking-widest">
            Device / Hostname
          </label>
          <button
            type="button"
            onClick={() => refetchDevices()}
            disabled={loadingDevices}
            className="text-[10px] text-terminal-cyan hover:text-terminal-green disabled:opacity-50 flex items-center gap-1"
          >
            {loadingDevices && <Spinner size="sm" />}
            <span>{loadingDevices ? 'Loading...' : '↻ Refresh'}</span>
          </button>
        </div>
        {tailscaleDevices.length > 0 ? (
          <form.AppField
            name="tailscale_service_host"
            children={(field: any) => (
              <field.SelectField
                label=""
                placeholder="Select a device..."
                options={tailscaleDevices.map((device) => ({
                  value: device.hostname,
                  label: `${device.hostname || device.name} (${device.addresses[0] || 'no IP'}) - ${device.online ? '🟢 Online' : '🔴 Offline'}`,
                }))}
                required
              />
            )}
          />
        ) : (
          <form.AppField
            name="tailscale_service_host"
            children={(field: any) => (
              <field.TextField
                label=""
                placeholder="Device hostname"
                required
              />
            )}
          />
        )}
      </div>

      <div className="grid md:grid-cols-3 gap-4">
        <form.AppField
          name="tailscale_service_protocol"
          children={(field: any) => (
            <field.SelectField
              label="Protocol"
              options={[
                { value: 'http', label: 'HTTP' },
                { value: 'https', label: 'HTTPS' },
                { value: 'tcp', label: 'TCP' },
              ]}
            />
          )}
        />
        <form.AppField
          name="tailscale_service_port"
          children={(field: any) => (
            <field.NumberField label="Port" min={1} max={65535} required />
          )}
        />
        <form.Subscribe
          selector={(state: any) => state.values.tailscale_service_protocol}
        >
          {(protocol: string) =>
            (protocol === 'http' || protocol === 'https') && (
              <form.AppField
                name="tailscale_service_path"
                children={(field: any) => (
                  <field.TextField label="Path" placeholder="/" />
                )}
              />
            )
          }
        </form.Subscribe>
      </div>

      {!hasTailscale && (
        <div className="p-3 bg-terminal-yellow/10 border border-terminal-yellow/30 rounded text-[10px] text-terminal-yellow">
          Configure Tailscale API credentials in Settings before using this
          check type
        </div>
      )}
    </div>
  );
}
