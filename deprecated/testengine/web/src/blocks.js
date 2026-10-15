// This catalog mirrors the engine's BlockMeta: the keys a block reads/writes,
// its default config, and the rules pre-filled onto new instances. For now it's
// duplicated here; the clean long-term move is to have the engine serve this
// over its (future) HTTP endpoint so there's one source of truth.
export const BLOCKS = {
  login: {
    label: 'Login',
    color: '#34d399',
    reads: [],
    writes: ['auth_token'],
    defaultConfig: { username: 'alice', shouldSucceed: true },
    defaultRules: [],
  },
  http_request: {
    label: 'HTTP request',
    color: '#38bdf8',
    reads: [],
    writes: ['resp_status', 'resp_body'],
    defaultConfig: { method: 'GET', url: 'https://example.com' },
    defaultRules: [],
  },
  assert: {
    label: 'Assert',
    color: '#a78bfa',
    reads: ['resp_status', 'resp_body'],
    writes: [],
    defaultConfig: { kind: 'status_equals', expected: 200 },
    defaultRules: [],
  },
  buy: {
    label: 'Buy',
    color: '#fb7185',
    reads: ['auth_token'],
    writes: [],
    defaultConfig: { item: 'widget' },
    defaultRules: [
      { kind: 'requires_key', key: 'auth_token' },
      { kind: 'requires_block_upstream', block: 'login' },
    ],
  },
  log: {
    label: 'Log',
    color: '#94a3b8',
    reads: [],
    writes: [],
    defaultConfig: { message: 'message' },
    defaultRules: [],
  },
  load: {
    label: 'Load',
    color: '#fbbf24',
    reads: [],
    writes: ['load_total', 'load_passed', 'load_failed', 'load_p50_ms', 'load_p95_ms'],
    defaultConfig: {
      workers: 20,
      iterations: 50,
      subgraph: { start: '', nodes: [], edges: [] },
    },
    defaultRules: [],
  },
};
