const mfersafeSettings = { rpc: "http://localhost:10545" };

chrome.storage.onChanged.addListener((changes, namespace) => {
  mfersafeSettings.rpc = changes.key.newValue;
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
