//////// BASE FUNCTION ////////
const resopnseDict = {};
const fetchWeb3 = function (request) {
  //   console.log("from injected: ", request);
  window.postMessage(
    {
      type: "FROM_PAGE",
      text: "hello from injected script",
      request: request,
    },
    "*"
  );

  return new Promise((resolve, reject) => {
    var requestKey = JSON.stringify(request);
    resopnseDict[requestKey] = resolve; // resolved by response
  });
};

window.addEventListener(
  "message",
  (event) => {
    if (event.source != window) {
      return;
    }
    if (event.data.type && event.data.type == "TO_PAGE") {
      var blob = new Blob([JSON.stringify(event.data.response)], {
        type: "application/json",
      });
      var newResponse = new Response(blob);
      var requestKey = JSON.stringify(event.data.request);
      resopnseDict[requestKey](newResponse); // resolve
      delete resopnseDict.requestKey; // free dict
    }
  },
  false
);

//////// INTERCEPT FETCH ////////
// https://stackoverflow.com/questions/45425169/intercept-fetch-api-requests-and-responses-in-javascript
const { fetch: origFetch } = window;
window.fetch = async (...args) => {
  if (args.length <= 1 || args[1].body === undefined) {
    return await origFetch(...args);
  }
  var jsonStr;
  var requestBody;
  try {
    if (typeof args[1].body === "string") {
      jsonStr = args[1].body;
    } else {
      jsonStr = new TextDecoder().decode(args[1].body);
    }
    requestBody = JSON.parse(jsonStr);
    if (
      requestBody.method === undefined ||
      requestBody.method.indexOf("eth_") < 0
    ) {
      return await origFetch(...args);
    }
  } catch (e) {
    // console.log("error:", e, args);
    return await origFetch(...args);
  }
  //   console.log(requestBody);
  return fetchWeb3(requestBody);
};

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
