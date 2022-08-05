//////// EIP1193 Provider ////////
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
  
      this.id = this.id + 1;
      const request = {
        jsonrpc: "2.0",
        id: this.id,
        method: args.method,
        //https://github.com/aklinkert/js-json-rpc-client/blob/master/src/index.js
        params: Array.isArray(args.params) ? args.params : [args.params],
      };
  
      return fetchWeb3(request)
        .then((content) => content.json())
        .then((result) => result.result);
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
  
    removeListener(event, listener) {
      console.log("removeListener:", event, listener);
    }
  
    removeAllListeners(args) {
      console.log("removeAllListeners:", args);
    }
  
    isMetaMask = true;
  
    chainId = "0x00";
  
    selectedAddress = "0x0000000000000000000000000000000000000000";
  
    constructor() {
      this.id = 0;
      this.request({ method: "eth_chainId", params: [] }).then((response) => {
        console.log("chainId =", response);
        this.chainId = response;
        this.netVersion = parseInt(response).toString();
      });
      this.request({ method: "eth_requestAccounts", params: [] }).then(
        (response) => {
          console.log("default address", response);
          this.selectedAddress = response[0];
        }
      );
    }
  }
  
  window.ethereum = new EIP1193Provider();
  