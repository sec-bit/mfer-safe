/*global chrome*/
export function docall(cmd, params) {
  var body = {
    jsonrpc: "2.0",
    id: 123,
    method: cmd,
    params: params,
  };

  if (!chrome.storage) {
    chrome.storage = {
      local: {
        get: (args, callback) =>
          new Promise((resolve, reject) => {
            callback({ "apesafer-rpc": "http://127.0.0.1:10545" });
          }),
        set: (args) => console.log(args),
      },
    };
  }

  // https://stackoverflow.com/questions/37700051/chrome-extension-is-there-any-way-to-make-chrome-storage-local-get-return-so
  function getData(sKey) {
    return new Promise(function (resolve, reject) {
      chrome.storage.local.get(sKey, function (items) {
        if (chrome.runtime && chrome.runtime.lastError) {
          console.error(chrome.runtime.lastError.message);
          reject(chrome.runtime.lastError.message);
        } else {
          resolve(items[sKey]);
        }
      });
    });
  }

  return getData(["apesafer-rpc"]).then(function (rpcURL) {
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
  });
}
