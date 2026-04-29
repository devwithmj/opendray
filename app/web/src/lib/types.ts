// ── Sessions ────────────────────────────────────────────────

export type SessionState = 'pending' | 'running' | 'idle' | 'ended'

export interface Session {
  id: string
  name?: string
  provider_id: string
  cwd: string
  args: string[]
  state: SessionState
  pid?: number
  started_at: string
  ended_at?: string
  exit_code?: number
}

export interface CreateSessionRequest {
  provider_id: string
  cwd: string
  name?: string
  args?: string[]
}

// ── Catalog (providers) ─────────────────────────────────────

export type ConfigFieldType =
  | 'string'
  | 'number'
  | 'boolean'
  | 'select'
  | 'secret'
  | 'args'

export interface ConfigField {
  key: string
  label: string
  label_zh?: string
  type: ConfigFieldType
  group?: string
  default?: unknown
  options?: string[]
  placeholder?: string
  description?: string
  description_zh?: string
  envVar?: string
  cliFlag?: string
  cliValue?: boolean
  dependsOn?: string
  dependsVal?: unknown
}

export interface ProviderManifest {
  id: string
  displayName: string
  displayName_zh?: string
  description: string
  description_zh?: string
  icon: string
  version: string
  kind: 'cli' | 'shell'
  executable: string
  defaultArgs?: string[]
  capabilities: {
    supportsResume: boolean
    supportsStream: boolean
    supportsImages: boolean
    supportsMcp: boolean
  }
  configSchema?: ConfigField[]
}

export interface Provider {
  manifest: ProviderManifest
  manifest_hash: string
  config: Record<string, unknown>
  enabled: boolean
}

// ── Channels ────────────────────────────────────────────────

export interface Channel {
  id: string
  kind: string
  config: Record<string, unknown>
  enabled: boolean
  running: boolean
}

export interface CreateChannelRequest {
  kind: string
  config: Record<string, unknown>
  enabled: boolean
}

// ── Integrations ────────────────────────────────────────────

export type IntegrationHealth =
  | 'unknown'
  | 'healthy'
  | 'degraded'
  | 'unhealthy'

export interface Integration {
  id: string
  name: string
  base_url: string
  route_prefix: string
  scopes: string[]
  version?: string
  enabled: boolean
  health_status: IntegrationHealth
  health_payload?: Record<string, unknown>
  health_last_seen?: string
  created_at: string
  rotated_at?: string
}

export interface RegisterIntegrationRequest {
  name: string
  base_url: string
  route_prefix: string
  scopes?: string[]
  version?: string
}

export interface RegisterIntegrationResult {
  integration: Integration
  api_key: string
}

export const ALL_SCOPES = [
  'session:read',
  'session:create',
  'session:input',
  'channel:send',
  'channel:receive',
  'event:subscribe:session.*',
  'event:subscribe:channel.*',
  'event:subscribe:integration.*',
  'provider:read',
] as const

export type Scope = (typeof ALL_SCOPES)[number] | string
