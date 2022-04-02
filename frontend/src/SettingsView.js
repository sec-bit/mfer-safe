/*global chrome*/
import React from "react";
import Button from "@material-ui/core/Button";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import { docall } from "./utils.js";
import SaveIcon from "@mui/icons-material/Save";
import FaceRetouchingNaturalIcon from "@mui/icons-material/FaceRetouchingNatural";
import PrintIcon from "@mui/icons-material/Print";
import InputAdornment from "@mui/material/InputAdornment";
import IconButton from "@mui/material/IconButton";

class SettingsView extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      rpc: "ws://127.0.0.1:9546",
    };

    this.handleRPCChange = this.handleRPCChange.bind(this);
    this.handleAccountChange = this.handleAccountChange.bind(this);
    this.handleFaucetReceiverChange =
      this.handleFaucetReceiverChange.bind(this);
  }

  save() {
    console.log("save state:", this.state.rpc);
    window.api.send("settings", { setrpc: this.state.rpc });
  }

  impersonate() {
    docall("ape_impersonate", [this.state.impersonatedAccount]);
  }

  printMoney() {
    docall("ape_printMoney", [this.state.faucetReceiver]);
  }

  handleRPCChange(event) {
    console.log("event:", event);
    this.setState({ rpc: event.target.value });
  }

  handleAccountChange(event) {
    console.log("event:", event);
    this.setState({ impersonatedAccount: event.target.value });
  }

  handleFaucetReceiverChange(event) {
    console.log("event:", event);
    this.setState({ faucetReceiver: event.target.value });
  }

  render() {
    return (
      <div className="calldata-text">
        <Box
          component="form"
          sx={{
            "& .MuiTextField-root": { m: 1, width: "400px" },
          }}
          noValidate
          autoComplete="on"
        >
          <div>
            <TextField
              value={this.state.impersonatedAccount}
              onChange={this.handleAccountChange}
              label="Impersonated Account"
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      edge="end"
                      color="primary"
                      onClick={() => this.impersonate()}
                    >
                      <FaceRetouchingNaturalIcon />
                    </IconButton>
                  </InputAdornment>
                ),
              }}
            />
          </div>
          <div>
            <TextField
              value={this.state.faucetReceiver}
              onChange={this.handleFaucetReceiverChange}
              label="Void Ether"
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      edge="end"
                      color="primary"
                      onClick={() => this.printMoney()}
                    >
                      <PrintIcon />
                    </IconButton>
                  </InputAdornment>
                ),
              }}
            />
          </div>
          <div>
            <TextField
              value={this.state.rpc}
              onChange={this.handleRPCChange}
              label="ApeSafer RPC"
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      edge="end"
                      color="primary"
                      onClick={() => this.save()}
                    >
                      <SaveIcon />
                    </IconButton>
                  </InputAdornment>
                ),
              }}
            />
            {/* <Button onClick={() => this.save()}>💾Save</Button> */}
          </div>
        </Box>
      </div>
    );
  }
}

export default SettingsView;
