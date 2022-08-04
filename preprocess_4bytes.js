const fs = require("fs");
const path = require("path");
const sigFolder = "4bytes/signatures";
const sigHashes = fs.readdirSync(sigFolder);
var processedSignature = {};
process.stdout.write("Processing signatures...\n");
sigHashes.forEach((name,idx) => {
  var result = fs.readFileSync(path.join(sigFolder, name));
  processedSignature[name] = result.toString().replace("\n", ";");
  process.stdout.write("("+idx+"/"+sigHashes.length+")"+"\r");
});
process.stdout.write("\nDone\n");

fs.writeFileSync(
  path.join("mfer-safe-desktop-app", "src", "functionSignatures.json"),
  JSON.stringify(processedSignature)
);
