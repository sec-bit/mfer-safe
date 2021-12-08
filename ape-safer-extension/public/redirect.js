(function () {
  const needRedir = {};
  const needExclude = {};
  const networkFilters = {
    urls: ["*://*/*"],
  };

  var apesafer_server = "http://127.0.0.1:10545/";

  chrome.storage.local.get(["apesafer-rpc"], (items) => {
    if (items["apesafer-rpc"] !== undefined) {
      apesafer_server = items["apesafer-rpc"];
    }

    console.log("redirect web3 request to:", apesafer_server);

    needExclude[apesafer_server] = true;

    chrome.webRequest.onBeforeRequest.addListener(
      (details) => {
        if (needRedir[details.url]) {
          return { redirectUrl: apesafer_server };
        }

        if (needExclude[details.url]) {
          return;
        }

        if (details.url !== apesafer_server && details.method === "POST") {
          try {
            var ret = details.requestBody.raw
              .map(function (data) {
                return String.fromCharCode.apply(
                  null,
                  new Uint8Array(data.bytes)
                );
              })
              .join("");
            console.log(ret);
            var method = JSON.parse(ret).method;
            if (method.indexOf("eth_") == 0) {
              console.log("redir: ", details.url, " to: ", apesafer_server);
              needRedir[details.url] = true;
              return { redirectUrl: apesafer_server };
            }
          } catch (e) {
            console.log(e);
            console.log(details);
            needExclude[details.url] = true;
            return;
          }
        }
      },
      networkFilters,
      ["blocking", "requestBody"]
    );
  })();
});
