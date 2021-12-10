import { useState, useEffect } from "react";
import { docall } from "./utils.js";
import TextField from "@mui/material/TextField";
import ReactJson from "react-json-view";

function TraceView() {
  const [callTrace, setCallTrace] = useState({});
  const [fullTrace, setFullTrace] = useState({});
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
              setCallTrace({ err: "Trace not found" });
            } else {
              const traceJSON = Buffer.from(
                traceLog.data.replace("0x", ""),
                "hex"
              ).toString();
              setCallTrace(JSON.parse(traceJSON));
            }
          } else {
            setCallTrace({ err: "Trace not found" });
          }
        },
        (error) => {
          console.log(error);
        }
      );
  }, []);

  useEffect(() => {
    docall("debug_traceTransaction", [txhash])
      .then((res) => res.json())
      .then(
        (result) => {
          if (result.hasOwnProperty("result")) {
            const traceResult = result.result;
            const fullTraceStr = JSON.stringify(traceResult, null, 2);
            setFullTrace(fullTraceStr);
          } else {
            setFullTrace(JSON.stringify({ err: "Trace not found" }));
          }
        },
        (error) => {
          console.log(error);
        }
      );
  }, []);

  return (
    <div style={{ textAlign: "left" }}>
      <ReactJson
        src={callTrace}
        displayDataTypes={false}
        enableClipboard={false}
      />
      <TextField
        style={{ textAlign: "left" }}
        value={fullTrace}
        multiline
        rows={500}
        fullWidth
      />
    </div>
  );
}

export default TraceView;
