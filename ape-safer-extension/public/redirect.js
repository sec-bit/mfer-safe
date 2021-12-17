(function () {
  const needRedir = {};
  const needExclude = {};
  const networkFilters = {
    urls: ["<all_urls>"],
  };

  var apesafer_server = "http://127.0.0.1:10545/";

  chrome.storage.onChanged.addListener(function (changes, namespace) {
    console.log(changes, namespace);
    apesafer_server = changes["apesafer-rpc"].newValue;
    console.log("redir server updated, new rpc:", apesafer_server);
    needExclude[apesafer_server] = true;
  });
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

        if (details.url.indexOf(apesafer_server) >= 0) {
          needExclude[details.url] = true;
          return;
        }

        if (needExclude[details.url]) {
          return;
        }

        if (details.url !== apesafer_server) {
          try {
            var ret = details.requestBody.raw
              .map(function (data) {
                return String.fromCharCode.apply(
                  null,
                  new Uint8Array(data.bytes)
                );
              })
              .join("");
            var method = JSON.parse(ret).method
              ? JSON.parse(ret).method
              : JSON.parse(ret)[0].method;

            if (method.indexOf("eth_") == 0) {
              console.log("redir: ", details.url, " to: ", apesafer_server);
              needRedir[details.url] = true;
              return { redirectUrl: apesafer_server };
            }
          } catch (e) {
            return;
          }
        }
      },
      networkFilters,
      ["blocking", "requestBody"]
    );
  });
})();
