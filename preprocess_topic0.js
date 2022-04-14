const fs = require("fs");
const path = require("path");
const sigFolder = "topic0/signatures";
const sigHashes = fs.readdirSync(sigFolder);
var processedSignature = {};
sigHashes.forEach((name) => {
  var result = fs.readFileSync(path.join(sigFolder, name));
  processedSignature[name] = result.toString().replace("\n", "");
});

fs.writeFileSync(
  path.join("frontend", "src", "eventSignatures.json"),
  JSON.stringify(processedSignature)
);
