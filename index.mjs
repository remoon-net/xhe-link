if (!WebAssembly.instantiateStreaming) {
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer();
    return await WebAssembly.instantiate(source, importObject);
  };
}

import { version } from "./package.json";
const defaultWasmUrl = `https://unpkg.com/@remoon.cn/xhe-link@${version}/xhe.wasm`;

export async function Init(wasmUrl = defaultWasmUrl) {
  const go = new Go();
  const { instance } = await WebAssembly.instantiateStreaming(
    fetch(wasmUrl),
    go.importObject
  );
  return { process: go.run(instance) };
}

// fetch("https://1.1.1.1/dns-query?name=lighthouse.lo.remoon.cn.remoon.net&type=AAAA", { headers: { accept: "application/dns-json" } });
