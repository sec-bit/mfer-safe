function saveRPCSettings() {
  var textfield = document.getElementById("rpc");
  chrome.storage.local.set({ rpcAddr: textfield.value }, function () {
    console.log("Value is set to " + textfield.value);
  });
}
const check_inj = document.querySelector("#cbox_inj");

function getAllSettings() {
  var textfield = document.getElementById("rpc");
  chrome.storage.local.get(["rpcAddr","inject"], function (result) {
    var rpcAddr = result.rpcAddr;
    console.log("Value currently is " + rpcAddr);
    textfield.value = rpcAddr || "http://localhost:10545";
    check_inj.checked = result.inject || false;
  });
}

const button = document.querySelector("button");
button.addEventListener("click", (event) => {
  saveRPCSettings();
});

check_inj.addEventListener("change", function (e) {
  console.log("Inject is set to " + e.currentTarget.checked);
  chrome.storage.local.set({ inject: e.currentTarget.checked }, function () {
  })
});

getAllSettings();