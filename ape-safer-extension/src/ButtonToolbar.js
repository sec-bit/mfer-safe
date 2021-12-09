/*global chrome*/
import React from "react";
import Button from "@material-ui/core/Button";
import Toolbar from "@mui/material/Toolbar";
import { docall } from "./utils.js";
import Box from "@mui/material/Box";
class ButtonToolbar extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      addresses: [],
      calltrace: {},
      approveInfo: {},
    };
    // this.handleInput = this.handleInput.bind(this);
  }

  simulate() {
    docall("ape_simulateSafeExec", [this.state.addresses])
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
                  execCalldata: result.result.multisendCalldata,
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

  render() {
    return (
      <Box>
        <Toolbar>
          <Button
            onClick={() => {
              docall("ape_reExecTxPool", []);
            }}
          >
            ♻️Re-exec
          </Button>
          <Button onClick={() => docall("ape_clearTxPool", [])}>
            🗑Clear TxPool
          </Button>
          <Button
            onClick={() => {
              window.open("?page=txs", "_blank");
            }}
          >
            🗒️View All Txs
          </Button>
        </Toolbar>
      </Box>
    );
  }
}

export default ButtonToolbar;
