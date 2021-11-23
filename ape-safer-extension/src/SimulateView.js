import React from "react";
import Button from "@material-ui/core/Button";
import AppBar from "@material-ui/core/AppBar";
import Badge from "@material-ui/core/Badge";
import FormatListBulletedIcon from "@mui/icons-material/FormatListBulleted";
import Toolbar from "@mui/material/Toolbar";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import FormGroup from "@mui/material/FormGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Checkbox from "@mui/material/Checkbox";
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
                calltrace: JSON.stringify(result.result.debugTrace, null, 2),
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
          <FormGroup>
            <FormControlLabel
              control={<Checkbox defaultChecked />}
              label="Label"
            />
            <FormControlLabel
              disabled={true}
              control={<Checkbox />}
              label="Disabled"
            />
          </FormGroup>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.safeAddr}
              label="To"
              defaultValue="Safe Contract Address"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.execCalldata}
              label="Exec Calldata"
              defaultValue="Multisend Call"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.approveHashCalldata}
              label="Approve CallData"
              defaultValue="ApproveHash Calldata"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.dataHash}
              label="Data Hash"
              defaultValue="eth_sign data"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.calltrace}
              label="Call Trace"
              defaultValue="Simulation Result"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
          <div>
            <TextField
              id="outlined-read-only-input"
              value={this.state.approveInfo.revertError}
              label="Result"
              defaultValue="Execution Result"
              InputProps={{
                readOnly: true,
              }}
            />
          </div>
        </Box>
      </div>
    );
  }
}

export default SimulateView;
