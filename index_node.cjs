require("./wrtc-polyfill");
require("./go_wasm_js/wasm_exec_node");
const fs = require("node:fs/promises");
const path = require("path");
const defaultWasmPath = path.join(module.path, "xhe.wasm");

exports.Init = async (wasmPath = defaultWasmPath) => {
  const go = new Go();
  const wasmBuf = await fs.readFile(wasmPath);
  const { instance } = await WebAssembly.instantiate(wasmBuf, go.importObject);
  return { process: go.run(instance) };
};
