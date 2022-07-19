// import { fetch } from '@tauri-apps/api/http';

import { invoke } from "@tauri-apps/api/tauri";

export async function getApeNodeArgs() {
  const args = await invoke("get_ape_node_args");
  return args;
}
export async function docall(cmd, params) {
  var body = {
    jsonrpc: "2.0",
    id: 123,
    method: cmd,
    params: params,
  };
  var args = await getApeNodeArgs();
  console.log("RPC:", args.listen_host_port);
  var rpcURL = "http://" + args.listen_host_port;
  var ret = fetch(rpcURL, {
    headers: {
      accept: "*/*",
      "content-type": "application/json",
    },
    body: JSON.stringify(body),
    method: "POST",
  });
  return ret;
}
