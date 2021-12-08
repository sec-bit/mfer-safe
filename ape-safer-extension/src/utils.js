/*global chrome*/
export function docall(cmd, params) {
  var body = {
    jsonrpc: "2.0",
    id: 123,
    method: cmd,
    params: params,
  };

  var rpcAddress = "http://127.0.0.1:10545/";

  chrome.storage.local.get(["apesafer-rpc"], (items) => {
    if (items["apesafer-rpc"] !== undefined) {
      rpcAddress = items["apesafer-rpc"];
    }

    var ret = fetch(rpcAddress, {
      headers: {
        accept: "*/*",
        "content-type": "application/json",
      },
      referrerPolicy: "strict-origin-when-cross-origin",
      body: JSON.stringify(body),
      method: "POST",
      mode: "cors",
      credentials: "omit",
    });
    return ret;
  });
}
