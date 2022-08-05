function saveRPCSettings() {
  var textfield = document.getElementById("rpc");
  chrome.storage.local.set({ rpcAddr: textfield.value }, function () {
    console.log("Value is set to " + textfield.value);
  });
}

function getRPCSettings() {
  var textfield = document.getElementById("rpc");
  chrome.storage.local.get(["rpcAddr"], function (result) {
    var rpcAddr = result.rpcAddr;
    console.log("Value currently is " + rpcAddr);
    textfield.value = rpcAddr||"http://localhost:10545";
  });
}

const button = document.querySelector("button");

button.addEventListener("click", (event) => {
  saveRPCSettings();
});
getRPCSettings();
