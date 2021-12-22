import React from "react";
import Button from "@material-ui/core/Button";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import ReactJson from "react-json-view";
import { docall } from "./utils.js";

class SimulateView extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      addresses: ["0x0000000000000000000000000000000000000000"],
      calltrace: {},
      approveInfo: {},
      owners: [],
      threshold: 2,
    };
  }

  simulate() {
    docall("ape_simulateSafeExec", [[]])
      .then((res) => res.json())
      .then(
        (result) => {
          if (result.hasOwnProperty("result")) {
            if (result.result.hasOwnProperty("debugTrace")) {
              this.setState({
                calltrace: result.result.debugTrace,
                approveInfo: {
                  safeAddr: result.result.to,
                  approveHashCalldata: result.result.approveHashCallData,
                  dataHash: result.result.dataHash,
                  execCalldata: result.result.multisendCalldata,
                  revertError: result.result.revertError,
                },
              });
            }
          }
        },
        (error) => {
          this.setState({
            error,
          });
        }
      );
  }

  submitHandler = (e) => {
    e.preventDefault();
    this.setState({
      listItems: [...this.state.addresses, this.state.userInput],
      userInput: "",
    });
  };

  render() {
    return (
      <div className="calldata-text">
        <Box
          component="form"
          sx={{
            "& .MuiTextField-root": { m: 1, width: "400px" },
          }}
          noValidate
          autoComplete="off"
        >
          <div>
            <Button onClick={() => this.simulate()}>🙉Simulate</Button>
          </div>
          {/* <FormGroup>
            <FormControlLabel
              control={<Checkbox defaultChecked />}
              label="Label"
            />
            <FormControlLabel
              disabled={true}
              control={<Checkbox />}
              label="Disabled"
            />
          </FormGroup> */}
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.safeAddr || ""}
              label="To"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.execCalldata || ""}
              label="ExecTransaction Calldata"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.approveHashCalldata || ""}
              label="ApproveHash CallData"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.dataHash || ""}
              label="Data Hash"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.revertError || ""}
              label="Execution Result"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <ReactJson
              src={this.state.calltrace || {}}
              displayDataTypes={false}
              enableClipboard={false}
            />
          </div>
        </Box>
      </div>
    );
  }
}

export default SimulateView;
