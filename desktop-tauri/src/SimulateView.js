import { useState, useEffect } from "react";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Checkbox from "@mui/material/Checkbox";
import FormGroup from "@mui/material/FormGroup";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import FormControlLabel from "@mui/material/FormControlLabel";

import ReactJson from "react-json-view";
import { docall } from "./utils.js";

import AbiEventForm from "./AbiEventForm.js";
import Fieldset from "./FieldSet.js";

import eventSignatures from "./eventSignatures.json";


const simulate = (setTrace, participants) => {
  docall("ape_simulateSafeExec", [participants])
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
  });

  const [owners, setOwners] = useState({ owners: [], threshold: 0 });
  const [checked, setChecked] = useState({});
  const [participants, setParticipants] = useState([]);
  const [overrideSignature, setOverrideSignature] = useState({});
  const [overridedExecCallData, setOverridedExecCallData] = useState("");

  useEffect(() => {
    docall("ape_getSafeOwnersAndThreshold", [])
      .then((res) => res.json())
      .then((result) => {
        console.log(result);
        setOwners(result.result);
        var checked = {};
        result.result.owners.map((owner) => {
          checked[owner] = false;
        });
        setChecked(checked);
      })
      .catch((error) => {
        setOwners({ owners: [], threshold: -1 });
        console.log(error);
      });
  }, []);

  const handleChange = (event) => {
    var postState = {
      ...checked,
      [event.target.name]: event.target.checked,
    };
    setChecked(postState);
    var p = [];
    for (const [key, value] of Object.entries(postState)) {
      if (value) {
        p.push(key);
      }
    }
    setParticipants(p);
  };

  const error = participants.length !== owners.threshold;

  return (
    <div>
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
          <Box
            component="div"
            sx={{
              "& .MuiTypography-root": {
                fontFamily: "Monospace",
                fontSize: "16px",
              },
              // fontFamily: "body",
            }}
            noValidate
            autoComplete="off"
            margin="8px"
            display="flex"
          >
            <FormControl
              required
              error={error}
              component="fieldset"
              variant="standard"
            >
              <FormLabel component="legend">
                Select {owners.threshold} Participants
              </FormLabel>
              <FormGroup>
                {owners.owners.map((owner, idx) => {
                  return (
                    <FormControlLabel
                      key={idx}
                      control={
                        <Checkbox
                          checked={checked[owner]}
                          onChange={handleChange}
                          name={owner}
                        />
                      }
                      label={owner}
                    />
                  );
                })}
              </FormGroup>
            </FormControl>
          </Box>
          <div>
            <Button
              disabled={error}
              onClick={() => {
                simulate(setCallTrace, participants);
              }}
            >
              🙉Simulate
            </Button>
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
            value={overridedExecCallData || callTrace.approveInfo.execCalldata || ""}
            label="ExecTransaction Calldata"
            InputProps={{
              readOnly: true,
            }}
            variant="filled"
          />
          {participants.map((participant, idx) => {
            // console.log(participant);
            return <TextField
              label={participant}
              key={idx}
              helperText="Signature override"
              value={overrideSignature[participant] ||""}
              onChange={(e) => {
                var sig = e.target.value;
                var newOverridedSig = { ...overrideSignature, [participant]: sig };
                setOverrideSignature(newOverridedSig)
                // console.log("new overrided sig:",newOverridedSig);
                var toBeReplaced = callTrace.approveInfo.execCalldata
                for (const [p, sig] of Object.entries(newOverridedSig)) {
                  if (!sig.startsWith("0x")) {
                    setOverridedExecCallData("");
                    continue;
                  }
                  if (sig.length !== 132) {
                    setOverridedExecCallData("");
                    continue;
                  }
                  // console.log("sig:",sig, "p:",p);
                  // TODO: replace start at the end of the calldata
                  var searchString = "000000000000000000000000" + p.slice(2) + "000000000000000000000000000000000000000000000000000000000000000001"
                  toBeReplaced = toBeReplaced.replace(searchString, sig.slice(2))
                }
                setOverridedExecCallData(toBeReplaced);
              }
              }
              size="small" />;
          })}
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
        {callTrace.eventLogs.map((event, idx) => {
          var eventName = eventSignatures[event.topics[0].slice(2)];
          if (eventName === undefined) {
            eventName = "Topic Name Not Found";
          }
          event.name = eventName;
          return <AbiEventForm key={idx} event={event} />;
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
