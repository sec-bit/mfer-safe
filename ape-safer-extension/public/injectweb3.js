function codeToInject() {
  (function () {
    var localrpc = "http://127.0.0.1:10545";
    var ethereum = new CAIP.EthereumProvider(localrpc);
    window.ethereum = ethereum;
    console.log("eip1193 provider injected");
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
      script.text = payload + "\n+" + `(${fn.toString()})();`;
      document.documentElement.appendChild(script);
    });
}

embed(codeToInject);
