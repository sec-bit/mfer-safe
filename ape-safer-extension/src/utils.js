export function docall(cmd, params) {
  var body = {
    jsonrpc: "2.0",
    id: 123,
    method: cmd,
    params: params,
  };
  var ret = fetch("http://127.0.0.1:10545/", {
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
