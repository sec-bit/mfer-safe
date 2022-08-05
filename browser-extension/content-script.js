var s = document.createElement("script");
s.src = chrome.runtime.getURL("redir.js");
s.onload = function () {
  this.remove();
};
(document.head || document.documentElement).appendChild(s);
console.log("redir injected");

chrome.storage.local.get(["inject"], function (result) {
  if (result.inject) {
    var s = document.createElement("script");
    s.src = chrome.runtime.getURL("eip1193provider.js");
    s.onload = function () {
      this.remove();
    };
    (document.head || document.documentElement).appendChild(s);
    console.log("eip1193 provider injected");
  }
});

window.addEventListener(
  "message",
  (event) => {
    // We only accept messages from ourselves
    if (event.source != window) {
      return;
    }

    if (event.data.type && event.data.type == "FROM_PAGE") {
      var request = event.data.request;
      chrome.runtime.sendMessage(request, (response) => {
        window.postMessage(
          {
            type: "TO_PAGE",
            text: "hello from content script",
            request: request,
            response: response,
          },
          "*"
        );
      });
    }
  },
  false
);
