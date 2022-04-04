import { React, useState, useCallback } from "react";
// import Button from "@material-ui/core/Button";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import { docall } from "./utils.js";
import SaveIcon from "@mui/icons-material/Save";
import FaceRetouchingNaturalIcon from "@mui/icons-material/FaceRetouchingNatural";
import PrintIcon from "@mui/icons-material/Print";
import InputAdornment from "@mui/material/InputAdornment";
import IconButton from "@mui/material/IconButton";
import LanIcon from '@mui/icons-material/Lan';

export default function SettingsView() {
  const [web3Rpc, setWeb3RPC] = useState("ws://127.0.0.1:8546")
  const [listenHostPort, setListenHostPort] = useState("127.0.0.1:10545")
  const [faucetReceiver, setFaucetReceiver] = useState("")
  const [impersonatedAccount, setImpersonatedAccount] = useState("0x0000000000000000000000000000000000000000")

  const saveRPCSettings = useCallback(() => {
    window.api.send("settings", {
      setweb3rpc: web3Rpc,
      setlistenhostport: listenHostPort
    });
  }, [web3Rpc,listenHostPort]);

  const impersonate = useCallback(() => {
    docall("ape_impersonate", [impersonatedAccount]);
  }, [impersonatedAccount]);

  const printMoney = useCallback(() => {
    docall("ape_printMoney", [faucetReceiver]);
  }, [faucetReceiver])

  // handleRPCChange(event) {
  //   console.log("event:", event);
  //   this.setState({ rpc: event.target.value });
  // }

  // handleAccountChange(event) {
  //   console.log("event:", event);
  //   this.setState({ impersonatedAccount: event.target.value });
  // }

  // handleFaucetReceiverChange(event) {
  //   console.log("event:", event);
  //   this.setState({ faucetReceiver: event.target.value });
  // }

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
            value={impersonatedAccount}
            onChange={e => setImpersonatedAccount(e.target.value)}
            label="Impersonated Account"
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton
                    edge="end"
                    color="primary"
                    onClick={() => impersonate()}
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
            value={faucetReceiver}
            onChange={e => setFaucetReceiver(e.target.value)}
            label="Void Ether"
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton
                    edge="end"
                    color="primary"
                    onClick={() => printMoney()}
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
            value={web3Rpc}
            onChange={e => setWeb3RPC(e.target.value)}
            label="Upstream Web3 RPC"
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton
                    edge="end"
                    color="primary"
                    onClick={() => saveRPCSettings()}
                  >
                    <SaveIcon />
                  </IconButton>
                </InputAdornment>
              ),
            }}
          />
        </div>
        <div>
          <TextField
            value={listenHostPort}
            onChange={e => setListenHostPort(e.target.value)}
            label="ApeSafer Listen"
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton
                    edge="end"
                    color="primary"
                    onClick={() => saveRPCSettings()}
                  >
                    <LanIcon />
                  </IconButton>
                </InputAdornment>
              ),
            }}
          />
        </div>
      </Box>
    </div>
  );
}
