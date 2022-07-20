import { React, useState, useCallback, useEffect } from "react";
import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import { getApeNodeArgs, docall } from "./utils.js";
import SaveIcon from "@mui/icons-material/Save";
import ViewModuleIcon from "@mui/icons-material/ViewModule";
import FaceRetouchingNaturalIcon from "@mui/icons-material/FaceRetouchingNatural";
import PrintIcon from "@mui/icons-material/Print";
import InputAdornment from "@mui/material/InputAdornment";
import IconButton from "@mui/material/IconButton";
import LanIcon from "@mui/icons-material/Lan";
import MapIcon from "@mui/icons-material/Map";
import ConfirmationNumberIcon from "@mui/icons-material/ConfirmationNumber";
import MoreTimeIcon from "@mui/icons-material/MoreTime";

import { invoke } from "@tauri-apps/api/tauri";

function IconButtonTextField(props) {
  return (
    <TextField
      fullWidth
      value={props.state}
      onChange={(e) => props.setState(e.target.value)}
      label={props.label}
      InputProps={{
        endAdornment: (
          <InputAdornment position="end">
            <IconButton edge="end" color="primary" onClick={props.onClick}>
              <props.icon />
            </IconButton>
          </InputAdornment>
        ),
      }}
    />
  );
}

export default function SettingsView() {
  const [impersonatedAccount, setImpersonatedAccount] = useState(
    "0x0000000000000000000000000000000000000000"
  );
  const [faucetReceiver, setFaucetReceiver] = useState("");

  const [web3Rpc, setWeb3RPC] = useState("ws://127.0.0.1:8546");
  const [listenHostPort, setListenHostPort] = useState("127.0.0.1:10545");
  const [batchSize, setBatchSize] = useState(100);
  const [blockNumberDelta, setBlockNumberDelta] = useState(0);
  const [blockTimeDelta, setBlockTimeDelta] = useState(0);
  const [keyCacheFilePath, setKeyCacheFilePath] = useState("");

  useEffect(() => {
    getApeNodeArgs().then((args) => {
      // avoid Safari: "Fetch API cannot load due to access control checks" fill init arg
      setImpersonatedAccount(args.impersonated_account);

      setWeb3RPC(args.web3_rpc);
      setListenHostPort(args.listen_host_port);
      setKeyCacheFilePath(args.key_cache_file_path);
      setBatchSize(args.batch_size);
    });

    // avoid Safari: "Fetch API cannot load due to access control checks" override after ape-node is started
    docall("eth_requestAccounts", [])
      .then((res) => res.json())
      .then((result) => {
        setImpersonatedAccount(result.result[0]);
      });

    docall("ape_getBlockNumberDelta", [])
      .then((res) => res.json())
      .then((result) => {
        setBlockNumberDelta(result.result);
      });
  }, []);

  docall("ape_getTimeDelta", [])
    .then((res) => res.json())
    .then((result) => {
      setBlockTimeDelta(result.result);
    });

  const saveRPCSettings = useCallback(() => {
    let args = {
      apeNodeArgs: {
        impersonated_account: impersonatedAccount,
        web3_rpc: web3Rpc,
        listen_host_port: listenHostPort,
        key_cache_file_path: keyCacheFilePath,
        log_file_path: "", //empty string means stdout
        batch_size: Number(batchSize),
      },
    };
    console.log(args);
    invoke("restart_ape_node", args);
  }, [
    impersonatedAccount,
    web3Rpc,
    listenHostPort,
    keyCacheFilePath,
    batchSize,
  ]);

  const impersonate = useCallback(() => {
    docall("ape_impersonate", [impersonatedAccount]);
  }, [impersonatedAccount]);

  const printMoney = useCallback(() => {
    docall("ape_printMoney", [faucetReceiver]);
  }, [faucetReceiver]);

  const setBatch = useCallback(() => {
    docall("ape_setBatchSize", [Number(batchSize)]);
  }, [batchSize]);

  const setBNDelta = useCallback(() => {
    docall("ape_setBlockNumberDelta", [Number(blockNumberDelta)]);
  }, [blockNumberDelta]);

  const setBTDelta = useCallback(() => {
    docall("ape_setTimeDelta", [Number(blockTimeDelta)]);
  }, [blockTimeDelta]);

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
        <Stack
          direction="column"
          justifyContent="flex-end"
          alignItems="center"
          spacing={2}
          padding={2}
        >
          <IconButtonTextField
            state={impersonatedAccount}
            setState={setImpersonatedAccount}
            label="Impersonated Account"
            icon={FaceRetouchingNaturalIcon}
            onClick={() => impersonate()}
          />
          <IconButtonTextField
            state={faucetReceiver}
            setState={setFaucetReceiver}
            label="Void Ether"
            icon={PrintIcon}
            onClick={() => printMoney()}
          />
          <IconButtonTextField
            state={batchSize}
            setState={setBatchSize}
            label="Batch Size"
            icon={ViewModuleIcon}
            onClick={() => setBatch()}
          />
          <Stack direction="row" width="50%">
            <IconButtonTextField
              state={blockNumberDelta}
              setState={setBlockNumberDelta}
              label="Block Number Delta"
              icon={ConfirmationNumberIcon}
              onClick={() => setBNDelta()}
            />
            <IconButtonTextField
              state={blockTimeDelta}
              setState={setBlockTimeDelta}
              label="Block Time Delta"
              icon={MoreTimeIcon}
              onClick={() => setBTDelta()}
            />
          </Stack>

          <IconButtonTextField
            state={web3Rpc}
            setState={setWeb3RPC}
            label="Upstream Web3 RPC"
            icon={SaveIcon}
            onClick={() => saveRPCSettings()}
          />
          <IconButtonTextField
            state={listenHostPort}
            setState={setListenHostPort}
            label="ApeSafer Listen"
            icon={LanIcon}
            onClick={() => saveRPCSettings()}
          />
          <IconButtonTextField
            state={keyCacheFilePath}
            setState={setKeyCacheFilePath}
            label="Key Cache File Path"
            icon={MapIcon}
            onClick={() => saveRPCSettings()}
          />
        </Stack>
      </Box>
    </div>
  );
}
