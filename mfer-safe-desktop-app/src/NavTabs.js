import * as React from "react";
import Box from "@mui/material/Box";
import Tabs from "@mui/material/Tabs";
import Tab from "@mui/material/Tab";
import { Link } from "react-router-dom";
import { docall } from "./utils.js";

import SettingsIcon from "@mui/icons-material/Settings";
import ListIcon from "@mui/icons-material/List";
import AccountBalanceWalletIcon from "@mui/icons-material/AccountBalanceWallet";
import SpeedDial from "@mui/material/SpeedDial";
import SpeedDialIcon from "@mui/material/SpeedDialIcon";
import SpeedDialAction from "@mui/material/SpeedDialAction";
import ReplayIcon from "@mui/icons-material/Replay";
import DeleteForeverIcon from "@mui/icons-material/DeleteForever";
import ClearIcon from '@mui/icons-material/Clear';
import TerminalIcon from "@mui/icons-material/Terminal";

const actions = [
  {
    icon: <ReplayIcon />,
    name: "Re-Exec",
    onClick: () => {
      docall("mfer_reExecTxPool", []);
    },
  },
  {
    icon: <DeleteForeverIcon />,
    name: "Clear TxPool",
    onClick: () => {
      docall("mfer_clearTxPool", []);
    },
  },
  {
    icon: <ClearIcon />,
    name: "Clear Key Cache",
    onClick: () => {
      docall("mfer_clearKeyCache", []);
    },
  },
];
export default function NavTabs() {
  const [value, setValue] = React.useState(0);
  const handleChange = (event, newValue) => {
    setValue(newValue);
  };

  return (
    <Box sx={{ width: "100%" }}>
      <SpeedDial
        ariaLabel="SpeedDial basic example"
        sx={{ position: "absolute", top: 8, right: 8 }}
        icon={<SpeedDialIcon />}
        direction="down"
      >
        {actions.map((action) => (
          <SpeedDialAction
            key={action.name}
            icon={action.icon}
            tooltipTitle={action.name}
            onClick={action.onClick}
          />
        ))}
      </SpeedDial>
      <Tabs
        value={value}
        onChange={handleChange}
        centered
        aria-label="nav tabs example"
      >
        <Tab
          icon={<SettingsIcon />}
          label="Settings"
          component={Link}
          to={"/"}
        />
        <Tab
          icon={<ListIcon />}
          label="Transactions"
          component={Link}
          to={"/txs"}
        />
        <Tab
          icon={<AccountBalanceWalletIcon />}
          label="Gnosis Safe"
          component={Link}
          to={"/safemultisend"}
        />
        <Tab
          icon={<TerminalIcon />}
          label="Logs"
          component={Link}
          to={"/logs"}
        />
      </Tabs>
    </Box>
  );
}
