function saveRPCSettings() {
  var textfield = document.getElementById("rpc");
  chrome.storage.local.set({ key: textfield.value }, function () {
    console.log("Value is set to " + textfield.value);
  });
}

function getRPCSettings() {
  var textfield = document.getElementById("rpc");
  chrome.storage.local.get(["key"], function (result) {
    console.log("Value currently is " + result.key);
    textfield.value = result.key;
  });
}

const button = document.querySelector("button");

button.addEventListener("click", (event) => {
  saveRPCSettings();
});
getRPCSettings();
