import { rmSync } from "node:fs";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { type Socket, connect as connectTcp } from "node:net";
import { join } from "node:path";

import { WalletProviderError } from "./errors";
import { fetchWithTimeout } from "./http";

type JsonValue = unknown;

type StoredJson = {
  value: JsonValue;
  expiresAt: number | null;
};

export type WalletCopilotStorage = {
  kind: "memory" | "local_file" | "upstash_redis" | "redis";
  getJson<T>(key: string): Promise<T | null>;
  setJson<T>(
    key: string,
    value: T,
    options?: { ttlSeconds?: number },
  ): Promise<void>;
  deleteKey(key: string): Promise<void>;
  pushJson<T>(
    key: string,
    value: T,
    options?: { ttlSeconds?: number },
  ): Promise<void>;
  popJson<T>(key: string): Promise<T | null>;
  increment(key: string, options?: { ttlSeconds?: number }): Promise<number>;
};

const memoryStore = new Map<string, StoredJson>();
let storage: WalletCopilotStorage | null = null;

export function getWalletCopilotStorage(): WalletCopilotStorage {
  if (storage) {
    return storage;
  }

  const redisUrl = process.env.UPSTASH_REDIS_REST_URL?.trim();
  const redisToken = process.env.UPSTASH_REDIS_REST_TOKEN?.trim();
  if (redisUrl && redisToken) {
    storage = createUpstashRedisStorage(redisUrl, redisToken);
    return storage;
  }

  const tcpRedisUrl = process.env.REDIS_URL?.trim();
  if (tcpRedisUrl) {
    storage = createRedisUrlStorage(tcpRedisUrl);
    return storage;
  }

  if (process.env.NODE_ENV !== "production") {
    storage = createLocalFileStorage();
    return storage;
  }

  storage = createMemoryStorage();
  return storage;
}

export async function checkWalletCopilotStorageHealth(): Promise<{
  ok: boolean;
  storage: WalletCopilotStorage["kind"];
  redis_configured: boolean;
  redis_url_configured: boolean;
  upstash_redis_configured: boolean;
  error: string | null;
}> {
  const selectedStorage = getWalletCopilotStorage();
  const healthKey = `qorvi:wallet-copilot-health:${Date.now()}`;
  try {
    await selectedStorage.setJson(healthKey, { ok: true }, { ttlSeconds: 30 });
    const value = await selectedStorage.getJson<{ ok: boolean }>(healthKey);
    await selectedStorage.deleteKey(healthKey);
    return {
      ok: value?.ok === true,
      storage: selectedStorage.kind,
      redis_configured: isAnyRedisConfigured(),
      redis_url_configured: isRedisUrlConfigured(),
      upstash_redis_configured: isUpstashRedisConfigured(),
      error: null,
    };
  } catch (error) {
    return {
      ok: false,
      storage: selectedStorage.kind,
      redis_configured: isAnyRedisConfigured(),
      redis_url_configured: isRedisUrlConfigured(),
      upstash_redis_configured: isUpstashRedisConfigured(),
      error: error instanceof Error ? error.message : "Storage health failed.",
    };
  }
}

export function resetWalletCopilotStorageForTests(): void {
  storage = null;
  memoryStore.clear();
  rmSync(localFileStorageDirectory(), { force: true, recursive: true });
}

function isUpstashRedisConfigured(): boolean {
  return Boolean(
    process.env.UPSTASH_REDIS_REST_URL?.trim() &&
      process.env.UPSTASH_REDIS_REST_TOKEN?.trim(),
  );
}

function isRedisUrlConfigured(): boolean {
  return Boolean(process.env.REDIS_URL?.trim());
}

function isAnyRedisConfigured(): boolean {
  return isUpstashRedisConfigured() || isRedisUrlConfigured();
}

function createMemoryStorage(): WalletCopilotStorage {
  return {
    kind: "memory",
    async getJson<T>(key: string): Promise<T | null> {
      const entry = memoryStore.get(key);
      if (!entry) {
        return null;
      }
      if (entry.expiresAt && entry.expiresAt <= Date.now()) {
        memoryStore.delete(key);
        return null;
      }
      return entry.value as T;
    },
    async setJson<T>(
      key: string,
      value: T,
      options?: { ttlSeconds?: number },
    ): Promise<void> {
      memoryStore.set(key, {
        value,
        expiresAt: options?.ttlSeconds
          ? Date.now() + options.ttlSeconds * 1000
          : null,
      });
    },
    async deleteKey(key: string): Promise<void> {
      memoryStore.delete(key);
    },
    async pushJson<T>(
      key: string,
      value: T,
      options?: { ttlSeconds?: number },
    ): Promise<void> {
      const current = await this.getJson<T[]>(key);
      await this.setJson(key, [...(current ?? []), value], options);
    },
    async popJson<T>(key: string): Promise<T | null> {
      const current = await this.getJson<T[]>(key);
      if (!current?.length) {
        return null;
      }
      const [value, ...rest] = current;
      await this.setJson(key, rest);
      return value ?? null;
    },
    async increment(
      key: string,
      options?: { ttlSeconds?: number },
    ): Promise<number> {
      const current = (await this.getJson<number>(key)) ?? 0;
      const next = current + 1;
      await this.setJson(key, next, options);
      return next;
    },
  };
}

function createLocalFileStorage(): WalletCopilotStorage {
  const directory = localFileStorageDirectory();

  return {
    kind: "local_file",
    async getJson<T>(key: string): Promise<T | null> {
      try {
        const raw = await readFile(filePathForKey(directory, key), "utf8");
        const entry = JSON.parse(raw) as StoredJson;
        if (entry.expiresAt && entry.expiresAt <= Date.now()) {
          await this.deleteKey(key);
          return null;
        }
        return entry.value as T;
      } catch {
        return null;
      }
    },
    async setJson<T>(
      key: string,
      value: T,
      options?: { ttlSeconds?: number },
    ): Promise<void> {
      await mkdir(directory, { recursive: true });
      const entry: StoredJson = {
        value,
        expiresAt: options?.ttlSeconds
          ? Date.now() + options.ttlSeconds * 1000
          : null,
      };
      await writeFile(filePathForKey(directory, key), JSON.stringify(entry));
    },
    async deleteKey(key: string): Promise<void> {
      await rm(filePathForKey(directory, key), { force: true });
    },
    async pushJson<T>(
      key: string,
      value: T,
      options?: { ttlSeconds?: number },
    ): Promise<void> {
      const current = await this.getJson<T[]>(key);
      await this.setJson(key, [...(current ?? []), value], options);
    },
    async popJson<T>(key: string): Promise<T | null> {
      const current = await this.getJson<T[]>(key);
      if (!current?.length) {
        return null;
      }
      const [value, ...rest] = current;
      await this.setJson(key, rest);
      return value ?? null;
    },
    async increment(
      key: string,
      options?: { ttlSeconds?: number },
    ): Promise<number> {
      const current = (await this.getJson<number>(key)) ?? 0;
      const next = current + 1;
      await this.setJson(key, next, options);
      return next;
    },
  };
}

function createUpstashRedisStorage(
  redisUrl: string,
  redisToken: string,
): WalletCopilotStorage {
  async function command<T>(parts: Array<string | number>): Promise<T> {
    const response = await fetchWithTimeout(redisUrl, {
      method: "POST",
      headers: {
        authorization: `Bearer ${redisToken}`,
        "content-type": "application/json",
      },
      body: JSON.stringify(parts),
      cache: "no-store",
    });
    const payload = (await response.json().catch(() => null)) as {
      result?: T;
      error?: string;
    } | null;

    if (!response.ok || payload?.error) {
      throw new WalletProviderError({
        code: "provider_unavailable",
        message:
          payload?.error ?? `Redis storage failed with ${response.status}.`,
        status: 503,
      });
    }

    return payload?.result as T;
  }

  return {
    kind: "upstash_redis",
    async getJson<T>(key: string): Promise<T | null> {
      const raw = await command<string | null>(["GET", key]);
      if (!raw) {
        return null;
      }
      return JSON.parse(raw) as T;
    },
    async setJson<T>(
      key: string,
      value: T,
      options?: { ttlSeconds?: number },
    ): Promise<void> {
      const serialized = JSON.stringify(value);
      if (options?.ttlSeconds) {
        await command<string>([
          "SET",
          key,
          serialized,
          "EX",
          options.ttlSeconds,
        ]);
        return;
      }
      await command<string>(["SET", key, serialized]);
    },
    async deleteKey(key: string): Promise<void> {
      await command<number>(["DEL", key]);
    },
    async pushJson<T>(
      key: string,
      value: T,
      options?: { ttlSeconds?: number },
    ): Promise<void> {
      await command<number>(["RPUSH", key, JSON.stringify(value)]);
      if (options?.ttlSeconds) {
        await command<number>(["EXPIRE", key, options.ttlSeconds]);
      }
    },
    async popJson<T>(key: string): Promise<T | null> {
      const raw = await command<string | null>(["LPOP", key]);
      if (!raw) {
        return null;
      }
      return JSON.parse(raw) as T;
    },
    async increment(
      key: string,
      options?: { ttlSeconds?: number },
    ): Promise<number> {
      const next = await command<number>(["INCR", key]);
      if (options?.ttlSeconds && next === 1) {
        await command<number>(["EXPIRE", key, options.ttlSeconds]);
      }
      return next;
    },
  };
}

function createRedisUrlStorage(redisUrl: string): WalletCopilotStorage {
  async function command<T>(parts: Array<string | number>): Promise<T> {
    const parsed = parseRedisUrl(redisUrl);
    const socket = await connectRedisSocket(parsed);
    try {
      if (parsed.password) {
        const authParts = parsed.username
          ? ["AUTH", parsed.username, parsed.password]
          : ["AUTH", parsed.password];
        await sendRedisCommand(socket, authParts);
      }
      if (parsed.database) {
        await sendRedisCommand(socket, ["SELECT", parsed.database]);
      }
      return (await sendRedisCommand(socket, parts)) as T;
    } catch (error) {
      throw new WalletProviderError({
        code: "provider_unavailable",
        message:
          error instanceof Error
            ? `Redis storage failed: ${error.message}`
            : "Redis storage failed.",
        status: 503,
      });
    } finally {
      socket.destroy();
    }
  }

  return {
    kind: "redis",
    async getJson<T>(key: string): Promise<T | null> {
      const raw = await command<string | null>(["GET", key]);
      if (!raw) {
        return null;
      }
      return JSON.parse(raw) as T;
    },
    async setJson<T>(
      key: string,
      value: T,
      options?: { ttlSeconds?: number },
    ): Promise<void> {
      const serialized = JSON.stringify(value);
      if (options?.ttlSeconds) {
        await command<string>([
          "SET",
          key,
          serialized,
          "EX",
          options.ttlSeconds,
        ]);
        return;
      }
      await command<string>(["SET", key, serialized]);
    },
    async deleteKey(key: string): Promise<void> {
      await command<number>(["DEL", key]);
    },
    async pushJson<T>(
      key: string,
      value: T,
      options?: { ttlSeconds?: number },
    ): Promise<void> {
      await command<number>(["RPUSH", key, JSON.stringify(value)]);
      if (options?.ttlSeconds) {
        await command<number>(["EXPIRE", key, options.ttlSeconds]);
      }
    },
    async popJson<T>(key: string): Promise<T | null> {
      const raw = await command<string | null>(["LPOP", key]);
      if (!raw) {
        return null;
      }
      return JSON.parse(raw) as T;
    },
    async increment(
      key: string,
      options?: { ttlSeconds?: number },
    ): Promise<number> {
      const next = await command<number>(["INCR", key]);
      if (options?.ttlSeconds && next === 1) {
        await command<number>(["EXPIRE", key, options.ttlSeconds]);
      }
      return next;
    },
  };
}

type ParsedRedisUrl = {
  host: string;
  port: number;
  username: string;
  password: string;
  database: number;
};

function parseRedisUrl(redisUrl: string): ParsedRedisUrl {
  try {
    const parsed = new URL(redisUrl);
    if (parsed.protocol !== "redis:") {
      throw new Error("REDIS_URL must use redis://.");
    }
    const database = Number(parsed.pathname.replace("/", "") || 0);
    return {
      host: parsed.hostname,
      port: parsed.port ? Number(parsed.port) : 6379,
      username: decodeURIComponent(parsed.username),
      password: decodeURIComponent(parsed.password),
      database: Number.isFinite(database) && database > 0 ? database : 0,
    };
  } catch (error) {
    throw new WalletProviderError({
      code: "provider_unavailable",
      message:
        error instanceof Error ? error.message : "Invalid REDIS_URL value.",
      status: 503,
    });
  }
}

async function connectRedisSocket(parsed: ParsedRedisUrl): Promise<Socket> {
  return new Promise((resolve, reject) => {
    const socket = connectTcp({
      host: parsed.host,
      port: parsed.port,
    });
    socket.setTimeout(5_000);
    function cleanup() {
      socket.off("connect", onConnect);
      socket.off("timeout", onTimeout);
      socket.off("error", onError);
    }
    function onConnect() {
      cleanup();
      resolve(socket);
    }
    function onTimeout() {
      cleanup();
      socket.destroy();
      reject(new Error("connection timed out"));
    }
    function onError(error: Error) {
      cleanup();
      reject(error);
    }
    socket.once("connect", onConnect);
    socket.once("timeout", onTimeout);
    socket.once("error", onError);
  });
}

async function sendRedisCommand(
  socket: Socket,
  parts: Array<string | number>,
): Promise<unknown> {
  const payload = encodeRedisCommand(parts);
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];

    function cleanup() {
      socket.off("data", onData);
      socket.off("error", onError);
    }

    function onError(error: Error) {
      cleanup();
      reject(error);
    }

    function onData(chunk: Buffer) {
      chunks.push(chunk);
      const buffer = Buffer.concat(chunks);
      const parsed = tryParseRedisResponse(buffer);
      if (!parsed.complete) {
        return;
      }
      cleanup();
      if (parsed.error) {
        reject(new Error(parsed.error));
        return;
      }
      resolve(parsed.value);
    }

    socket.on("data", onData);
    socket.once("error", onError);
    socket.write(payload);
  });
}

function encodeRedisCommand(parts: Array<string | number>): string {
  const encoded = [`*${parts.length}`];
  for (const part of parts) {
    const value = String(part);
    encoded.push(`$${Buffer.byteLength(value)}`, value);
  }
  return `${encoded.join("\r\n")}\r\n`;
}

function tryParseRedisResponse(
  buffer: Buffer,
):
  | { complete: false }
  | { complete: true; value: unknown; error?: never }
  | { complete: true; value?: never; error: string } {
  try {
    const parsed = parseRedisResponse(buffer, 0);
    if (!parsed) {
      return { complete: false };
    }
    if (parsed.error) {
      return { complete: true, error: parsed.error };
    }
    return { complete: true, value: parsed.value };
  } catch (error) {
    return {
      complete: true,
      error: error instanceof Error ? error.message : "Invalid Redis response.",
    };
  }
}

function parseRedisResponse(
  buffer: Buffer,
  offset: number,
):
  | { value: unknown; nextOffset: number; error?: never }
  | { value?: never; nextOffset: number; error: string }
  | null {
  const prefix = buffer.toString("utf8", offset, offset + 1);
  if (!prefix) {
    return null;
  }

  if (prefix === "+" || prefix === "-" || prefix === ":") {
    const lineEnd = buffer.indexOf("\r\n", offset);
    if (lineEnd === -1) {
      return null;
    }
    const line = buffer.toString("utf8", offset + 1, lineEnd);
    if (prefix === "-") {
      return { nextOffset: lineEnd + 2, error: line };
    }
    return {
      nextOffset: lineEnd + 2,
      value: prefix === ":" ? Number(line) : line,
    };
  }

  if (prefix === "$") {
    const lineEnd = buffer.indexOf("\r\n", offset);
    if (lineEnd === -1) {
      return null;
    }
    const length = Number(buffer.toString("utf8", offset + 1, lineEnd));
    if (length === -1) {
      return { nextOffset: lineEnd + 2, value: null };
    }
    const bodyStart = lineEnd + 2;
    const bodyEnd = bodyStart + length;
    if (buffer.length < bodyEnd + 2) {
      return null;
    }
    return {
      nextOffset: bodyEnd + 2,
      value: buffer.toString("utf8", bodyStart, bodyEnd),
    };
  }

  if (prefix === "*") {
    const lineEnd = buffer.indexOf("\r\n", offset);
    if (lineEnd === -1) {
      return null;
    }
    const length = Number(buffer.toString("utf8", offset + 1, lineEnd));
    if (length === -1) {
      return { nextOffset: lineEnd + 2, value: null };
    }
    const values: unknown[] = [];
    let currentOffset = lineEnd + 2;
    for (let index = 0; index < length; index += 1) {
      const parsed = parseRedisResponse(buffer, currentOffset);
      if (!parsed) {
        return null;
      }
      if (parsed.error) {
        return parsed;
      }
      values.push(parsed.value);
      currentOffset = parsed.nextOffset;
    }
    return { nextOffset: currentOffset, value: values };
  }

  throw new Error(`Unsupported Redis response prefix: ${prefix}`);
}

function filePathForKey(directory: string, key: string): string {
  return join(directory, `${encodeURIComponent(key)}.json`);
}

function localFileStorageDirectory(): string {
  return join(process.cwd(), ".qorvi-wallet-copilot-store");
}
