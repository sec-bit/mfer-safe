import React from "react";
import Button from "@material-ui/core/Button";
import AppBar from "@material-ui/core/AppBar";
import Badge from "@material-ui/core/Badge";
import Toolbar from "@mui/material/Toolbar";
import SpeedDial from "@mui/material/SpeedDial";
import SpeedDialIcon from "@mui/material/SpeedDialIcon";
import SpeedDialAction from "@mui/material/SpeedDialAction";
import { docall } from "./utils.js";
import {
  Replay as ReplayIcon,
  PlayArrow as PlayArrowIcon,
  DeleteForever as DeleteForeverIcon,
  FormatListBulleted as FormatListBulletedIcon,
} from "@mui/icons-material";
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
        <Toolbar position="static">
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
        </Toolbar>
      </Box>
    );
  }
}

export default ButtonToolbar;
