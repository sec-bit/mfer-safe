function codeToInject() {
  (function () {
    var localrpc = "http://127.0.0.1:10545";
    var ethereum = new CAIP.EthereumProvider(localrpc);
    window.ethereum = ethereum;
    console.log("eip1193 provider injected, RPC:", localrpc);
  })();
}

function embed(fn) {
  fetch(chrome.extension.getURL("provider.js"))
    .then(function (response) {
      return response.blob();
    })
    .then((blob) => blob.text())
    .then(function (payload) {
      const script = document.createElement("script");

      var funcStr = `(${fn.toString()})();`;

      chrome.storage.local.get(["apesafer-rpc"], (items) => {
        if (items["apesafer-rpc"] !== undefined) {
          funcStr = funcStr.replace(
            `http://127.0.0.1:10545`,
            items["apesafer-rpc"]
          );
        }
        script.text = payload + "\n" + funcStr;
        document.documentElement.appendChild(script);
      });
    });
}

embed(codeToInject);
