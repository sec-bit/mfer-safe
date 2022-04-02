// All of the Node.js APIs are available in the preload process.
// It has the same sandbox as a Chrome extension.
const { contextBridge, ipcRenderer } = require("electron");
const fs = require("fs");
const path = require("path");

contextBridge.exposeInMainWorld("providerFetch", {
  fetch: (args) => ipcRenderer.invoke("eth:fetch", args),
});

const providerFilePath = path.join(__dirname, "eip1193provider.js");
const provider = fs.readFileSync(providerFilePath).toString();
var script = document.createElement("script");
script.text = provider;
document.appendChild(script);
console.log(script);
