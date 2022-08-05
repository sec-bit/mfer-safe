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

