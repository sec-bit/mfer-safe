import { useState } from "react";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import ReactJson from "react-json-view";
import { docall } from "./utils.js";

import AbiEventForm from "./AbiEventForm.js";
import Fieldset from "./FieldSet.js";

import eventSignatures from "./eventSignatures.json";

const simulate = (setTrace) => {
  docall("ape_simulateSafeExec", [[]])
    .then((res) => res.json())
    .then(
      (result) => {
        if (result.hasOwnProperty("result")) {
          if (result.result.hasOwnProperty("debugTrace")) {
            setTrace({
              calltrace: result.result.debugTrace,
              approveInfo: {
                safeAddr: result.result.to,
                approveHashCalldata: result.result.approveHashCallData,
                dataHash: result.result.dataHash,
                execCalldata: result.result.multisendCalldata,
                revertError: result.result.revertError,
              },
              eventLogs: result.result.eventLogs,
            });
          }
        }
      },
      (error) => {
        setTrace({
          error,
        });
      }
    );
};

function SimulateView() {
  const [callTrace, setCallTrace] = useState({
    addresses: ["0x0000000000000000000000000000000000000000"],
    calltrace: {},
    approveInfo: {},
    owners: [],
    eventLogs: [],
    // threshold: 2,
  });

  return (
    <div className="calldata-text">
      <Box
        component="div"
        sx={{
          "& .MuiTextField-root": { m: 1, width: "500px" },
        }}
        noValidate
        autoComplete="off"
        justifyContent="center"
        alignItems="center"
        display="flex"
      >
        <Stack>
          <div>
            <Button onClick={() => simulate(setCallTrace)}>🙉Simulate</Button>
          </div>
          <TextField
            id="outlined-read-only-input"
            value={callTrace.approveInfo.safeAddr || ""}
            label="To"
            InputProps={{
              readOnly: true,
            }}
          />
          <TextField
            id="outlined-read-only-input"
            multiline
            maxRows={4}
            value={callTrace.approveInfo.execCalldata || ""}
            label="ExecTransaction Calldata"
            InputProps={{
              readOnly: true,
            }}
            variant="filled"
          />
          <TextField
            id="outlined-read-only-input"
            multiline
            value={callTrace.approveInfo.approveHashCalldata || ""}
            label="ApproveHash CallData"
            InputProps={{
              readOnly: true,
            }}
            variant="filled"
          />
          <TextField
            id="outlined-read-only-input"
            multiline
            value={callTrace.approveInfo.dataHash || ""}
            label="Data Hash"
            InputProps={{
              readOnly: true,
            }}
            variant="filled"
          />
          <TextField
            id="outlined-read-only-input"
            value={callTrace.approveInfo.revertError || ""}
            label="Execution Result"
            InputProps={{
              readOnly: true,
            }}
            variant="filled"
          />
        </Stack>
      </Box>
      <Fieldset legend="Event Logs">
        {callTrace.eventLogs.map((event) => {
          var eventName = eventSignatures[event.topics[0].slice(2)];
          if (eventName === undefined) {
            eventName = "Topic Name Not Found";
          }
          event.name = eventName;
          return <AbiEventForm key={[event]} event={event} />;
        })}
      </Fieldset>
      <ReactJson
        src={callTrace.calltrace || {}}
        displayDataTypes={false}
        enableClipboard={true}
      />
    </div>
  );
}

export default SimulateView;
