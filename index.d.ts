export interface Config {
  log_level: "debug" | "info" | "warn" | "error";
  private_key: string;
  doh?: string;
  links?: string[];
  peers?: string[];
}

export interface XheLink {
  (config: Config): Promise<XheWireguard>;
}

export interface XheWireguard {
  ListenTCP(port?: number): Promise<TCPServer>;
  IpcGet(): Promise<string>;
  IpcSet(config: string): Promise<void>;
}

export interface Hono {
  fetch(req: Request): Response | Promise<Response>;
}

export interface TCPServer {
  Serve(): Promise<void>;
  ServeHTTP(hono: Hono): Promise<void>;
  Close(): Promise<void>;
  ServeReady(): boolean;
  ReverseProxy(path: string, remote: string): Promise<void>;
  HandleEval(path: string): void;
}

declare global {
  var XheLink: XheLink;
}

export const Init: (wasmUrl?: string) => Promise<any>;
