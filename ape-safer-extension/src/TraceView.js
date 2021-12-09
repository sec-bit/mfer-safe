import { useState, useEffect } from "react";
import { docall } from "./utils.js";
// import SyntaxHighlighter from "react-syntax-highlighter";
// import { docco } from "react-syntax-highlighter/dist/esm/styles/hljs";
import ReactJson from "react-json-view";

function TraceView() {
  const [trace, setTrace] = useState({});
  const searchParams = new URLSearchParams(window.location.search);
  const txhash = searchParams.get("txhash");
  console.log(txhash);
  useEffect(() => {
    docall("eth_getTransactionReceipt", [txhash])
      .then((res) => res.json())
      .then(
        (result) => {
          if (result.hasOwnProperty("result")) {
            const logs = result.result.logs;
            const traceLog = logs[logs.length - 1];
            if (
              !traceLog ||
              traceLog.address !== "0xa9e5afe700000000a9e5afe700000000a9e5afe7"
            ) {
              console.log("trace not found");
              setTrace({ err: "Trace not found" });
            } else {
              const traceJSON = Buffer.from(
                traceLog.data.replace("0x", ""),
                "hex"
              ).toString();
              const traceJSONStr = JSON.stringify(
                JSON.parse(traceJSON),
                null,
                2
              );
              setTrace(JSON.parse(traceJSON));
              // setTrace("aa");
            }
          } else {
            setTrace({ err: "Trace not found" });
          }
        },
        (error) => {
          console.log(error);
        }
      );
  }, []);

  return (
    <div style={{ textAlign: "left" }}>
      <ReactJson src={trace} displayDataTypes={false} enableClipboard={false} />
    </div>
  );
}

export default TraceView;
