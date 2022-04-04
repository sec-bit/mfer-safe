class EIP1193Provider {
  request(args) {
    if (args === undefined) {
      return;
    }

    switch (args.method) {
      case "eth_chainId":
        if (this.chainId !== "0x00") {
          return Promise.resolve(this.chainId);
        }
        break;
      case "net_version":
        if (this.netVersion !== "0") {
          return Promise.resolve(this.netVersion);
        }
        break;
    }

    if (args && args.params === undefined) {
      if (typeof args === "string" || args instanceof String) {
        args = { method: args };
      }
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
    const ret = window.providerFetch.fetch({
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
      method: "POST",
      mode: "cors",
      credentials: "omit",
    });
    return ret;
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

  on(event, listener) {
    console.log("on:", event, listener);
  }

  isMetaMask = true;

  chainId = "0x00";

  constructor() {
    this.id = 0;
    this.request({ method: "eth_chainId", params: [] }).then((response) => {
      console.log("chainId =", response);
      this.chainId = response;
      this.netVersion = parseInt(response).toString();
    });
  }
}

window.ethereum = new EIP1193Provider();