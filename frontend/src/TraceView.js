import { useState, useEffect } from "react";
import { docall } from "./utils.js";
import eventSignatures from "./eventSignatures.json";
import TextField from "@mui/material/TextField";
import ReactJson from "react-json-view";

function TraceView() {
  const [callTrace, setCallTrace] = useState({});
  const [events, setEvents] = useState();
  // const [fullTrace, setFullTrace] = useState({});
  const searchParams = new URLSearchParams(window.location.search);
  const txhash = searchParams.get("txhash");

  useEffect(() => {
    docall("eth_getTransactionReceipt", [txhash])
      .then((res) => res.json())
      .then(
        (result) => {
          if (result.hasOwnProperty("result")) {
            const logs = result.result.logs;
            const traceLog = logs[logs.length - 1];
            const txEvents = logs.slice(0, logs.length - 1).map((log) => {
              var eventName = "";
              if (log.topics.length > 0) {
                eventName = eventSignatures[log.topics[0].slice(2)];
                if (eventName === undefined) {
                  eventName = "Topic Name Not Found";
                }
              }
              // debugger;
              return {
                address: log.address,
                name: eventName,
                topics: log.topics,
                data: log.data,
              };
            });
            setEvents(txEvents);
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
  }, [txhash]);

  return (
    <div style={{ textAlign: "left" }}>
      <ReactJson
        src={callTrace}
        displayDataTypes={false}
        enableClipboard={false}
      />
      <ReactJson
        src={events}
        displayDataTypes={false}
        enableClipboard={false}
      />
    </div>
  );
}

export default TraceView;
