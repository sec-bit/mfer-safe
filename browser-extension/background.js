const mfersafeSettings = { rpc: "http://localhost:10545" };

chrome.storage.local.get(["rpcAddr"], function (result) {
  var rpcAddr = result.rpcAddr || "http://localhost:10545";
  console.log("rpcAddr init value: " + rpcAddr);
  mfersafeSettings.rpc = rpcAddr;
});

chrome.storage.onChanged.addListener((changes, namespace) => {
  console.log("rpc changes from:", mfersafeSettings.rpc, "to:", changes.rpcAddr.newValue);
  mfersafeSettings.rpc = changes.rpcAddr.newValue;
});

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (
    request === undefined ||
    request.method === undefined ||
    request.method.indexOf("eth_") < 0
  ) {
    sendResponse(false);
    return;
  }

  try {
    var reqBody = {
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(request),
      method: "POST",
    };

    fetch(mfersafeSettings.rpc, reqBody)
      .then((response) => response.json())
      .then((body) => {
        console.log("mfernode response:", body);
        sendResponse(body);
      });
  } catch (e) {
    console.log(request, e);
    sendResponse(false);
  }
  return true;
});
