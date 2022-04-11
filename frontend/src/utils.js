export function docall(cmd, params) {
  var body = {
    jsonrpc: "2.0",
    id: 123,
    method: cmd,
    params: params,
  };
  var rpcURL = "http://127.0.0.1:10545";
  var ret = fetch(rpcURL, {
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
}
