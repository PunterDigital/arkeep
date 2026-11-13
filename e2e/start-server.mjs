// Starts a built arkeep-server against a throwaway data dir and SQLite file,
// for Playwright's webServer. Never touches a real Arkeep install.
// ARKEEP_SERVER_BIN overrides the binary (default ../bin/arkeep-server[.exe]).
import { spawn } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { HTTP_PORT, GRPC_PORT } from './ports.mjs';

const here = fileURLToPath(new URL('.', import.meta.url));
const bin =
  process.env.ARKEEP_SERVER_BIN ??
  resolve(here, '..', 'bin', process.platform === 'win32' ? 'arkeep-server.exe' : 'arkeep-server');
const dir = mkdtempSync(join(tmpdir(), 'arkeep-e2e-'));

const child = spawn(bin, [], {
  stdio: 'inherit',
  env: {
    ...process.env,
    ARKEEP_HTTP_ADDR: `127.0.0.1:${HTTP_PORT}`,
    ARKEEP_GRPC_ADDR: `127.0.0.1:${GRPC_PORT}`,
    ARKEEP_DB_DRIVER: 'sqlite',
    ARKEEP_DB_DSN: join(dir, 'arkeep.db'),
    ARKEEP_DATA_DIR: join(dir, 'data'),
    // Test-only values for a throwaway database.
    ARKEEP_SECRET_KEY: 'e2e-secret-key-not-for-production-0123456789',
    ARKEEP_AGENT_SECRET: 'e2e-agent-secret-not-for-production',
    ARKEEP_TELEMETRY: 'false',
    ARKEEP_UPDATE_CHECK: 'false',
    ARKEEP_LOG_LEVEL: 'warn',
  },
});

const cleanup = () => {
  try {
    rmSync(dir, { recursive: true, force: true });
  } catch {
    // Windows can still hold the SQLite file for a moment; the OS temp dir is fine.
  }
};
child.on('exit', (code) => {
  cleanup();
  process.exit(code ?? 0);
});
for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => child.kill(signal));
}
