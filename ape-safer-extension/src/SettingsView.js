/*global chrome*/
import React from "react";
import Button from "@material-ui/core/Button";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import { docall } from "./utils.js";

class SettingsView extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      rpc: "http://127.0.0.1:10545",
    };

    chrome.storage.local.get(["apesafer-rpc"], (items) => {
      console.log("get items:", items);
      if (items["apesafer-rpc"] === undefined) {
        console.log("items undefined");
        var localrpc = "http://127.0.0.1:10545";
        chrome.storage.local.set({ "apesafer-rpc": localrpc }, function () {
          console.log("set apesafer rpc endpoint to localhost");
          items["apesafer-rpc"] = localrpc;
        });
      }
      console.log("items:", items, "val:", items["apesafer-rpc"]);
      this.setState({ rpc: items["apesafer-rpc"] });
    });

    this.handleRPCChange = this.handleRPCChange.bind(this);
    this.handleAccountChange = this.handleAccountChange.bind(this);
  }

  save() {
    console.log("save state:", this.state.rpc);
    chrome.storage.local.set(
      { "apesafer-rpc": this.state.rpc },
      function () {}
    );
  }

  impersonate() {
    docall("ape_impersonate", [this.state.impersonatedAccount]);
  }

  handleRPCChange(event) {
    console.log("event:", event);
    this.setState({ rpc: event.target.value });
  }
  handleAccountChange(event) {
    console.log("event:", event);
    this.setState({ impersonatedAccount: event.target.value });
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
            />
          </div>
          <Button onClick={() => this.impersonate()}>🎭Impersonate</Button>
          <div>
            <TextField
              value={this.state.rpc}
              onChange={this.handleRPCChange}
              label="ApeSafer RPC"
            />
          </div>
          <Button onClick={() => this.save()}>
            💾Save (Restart your browser please.)
          </Button>
        </Box>
      </div>
    );
  }
}

export default SettingsView;
