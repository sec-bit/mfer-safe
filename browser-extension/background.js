const apesaferSettings = { rpc: "http://localhost:10545" };

chrome.storage.onChanged.addListener((changes, namespace) => {
  apesaferSettings.rpc = changes.key.newValue;
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

    fetch(apesaferSettings.rpc, reqBody)
      .then((response) => response.json())
      .then((body) => {
        console.log("apenode response:", body);
        sendResponse(body);
      });
  } catch (e) {
    console.log(request, e);
    sendResponse(false);
  }
  return true;
});
