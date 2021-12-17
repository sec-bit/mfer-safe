function codeToInject() {
  (function () {
    class EIP1193Provider {
      request(args) {
        if (args === undefined) {
          return;
        }

        // switch (args.method) {
        //   case "eth_chainId":
        //     if (this.chainId !== "0x00") {
        //       return new Promise(function (resolve, reject) {
        //         resolve(this.chainId);
        //       });
        //     }
        //     break;
        //   case "net_version":
        //     if (this.netVersion !== "0") {
        //       return new Promise(function (resolve, reject) {
        //         resolve(this.netVersion);
        //       });
        //     }
        //     break;
        // }

        if (args && args.params === undefined) {
          args.params = [];
        }

        const id = this.id++;
        const body = {
          jsonrpc: "2.0",
          id: id,
          method: args.method,
          //https://github.com/aklinkert/js-json-rpc-client/blob/master/src/index.js
          params: Array.isArray(args.params) ? args.params : [args.params],
        };

        console.log("reqBody:", JSON.stringify(body));
        const ret = fetch(this.rpcURL, {
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
          },
          body: JSON.stringify(body),
          method: "POST",
          mode: "cors",
          credentials: "omit",
        });
        return ret.then((res) => res.json()).then((data) => data.result);
      }

      enable() {
        console.log("enable");
        return this.request({ method: "eth_accounts", params: [] });
      }

      send(args) {
        return this.request(args);
      }

      isConnected() {
        console.log("is connected");
        return true;
      }

      _metamask = {
        isUnlocked: function () {
          return true;
        },
      };

      isMetaMask = true;
      chainId = "0x00";

      constructor(rpcURL) {
        console.log(rpcURL);
        this.rpcURL = rpcURL;
        this.id = 0;
        this.request({ method: "eth_chainId", params: [] }).then((response) => {
          console.log("chainId =", response);
          this.chainId = response;
          this.netVersion = parseInt(response).toString();
        });
      }
    }

    var localrpc = "http://127.0.0.1:10545";
    var ethereum = new EIP1193Provider(localrpc);

    window.ethereum = ethereum;
    console.log("eip1193 provider injected, RPC:", localrpc);
  })();
}

function embed(fn) {
  const script = document.createElement("script");

  var funcStr = `(${fn.toString()})();`;

  chrome.storage.local.get(["apesafer-rpc"], (items) => {
    if (items["apesafer-rpc"] !== undefined) {
      funcStr = funcStr.replace(
        `http://127.0.0.1:10545`,
        items["apesafer-rpc"]
      );
    }
    // payload = "";
    script.text = funcStr;
    document.documentElement.appendChild(script);
  });
}

embed(codeToInject);
