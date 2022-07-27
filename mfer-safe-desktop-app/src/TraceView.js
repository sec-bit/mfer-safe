import { useState, useEffect } from "react";
import { Buffer } from "buffer";
import ReactJson from "react-json-view";

import { docall } from "./utils.js";
import eventSignatures from "./eventSignatures.json";
import { useParams } from "react-router-dom";

import AbiEventForm from "./AbiEventForm.js";
import Fieldset from "./FieldSet.js";

function TraceView() {
  const [callTrace, setCallTrace] = useState({});
  const [events, setEvents] = useState([{ name: "x", topics: [] }]);
  // const [fullTrace, setFullTrace] = useState({});
  let { txHash } = useParams();
  useEffect(() => {
    docall("eth_getTransactionReceipt", [txHash])
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
              traceLog.address !== "0x3fe75afe000000003fe75afe000000003fe75afe"
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
  }, [txHash]);

  return (
    <div style={{ textAlign: "left" }}>
      <Fieldset legend="Event Logs">
        {events.map((event) => {
          return <AbiEventForm key={[event]} event={event} />;
        })}
      </Fieldset>
      <ReactJson
        src={callTrace}
        displayDataTypes={false}
        enableClipboard={true}
      />
    </div>
  );
}

export default TraceView;
