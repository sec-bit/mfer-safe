// import React from "react";
import Button from "@material-ui/core/Button";
import Toolbar from "@mui/material/Toolbar";
import { docall } from "./utils.js";
import Box from "@mui/material/Box";

export default function ButtonToolbar() {
  return (
    <Box>
      <Toolbar>
        <Button
          onClick={() => {
            window.open("?page=home", "_self");
          }}
        >
          🏠Home
        </Button>
        <Button
          onClick={() => {
            window.open("?page=txs", "_self");
          }}
        >
          🗒️Txn List
        </Button>
        <Button
          onClick={() => {
            window.open("?page=safemultisend", "_self");
          }}
        >
          📦Gnosis Safe
        </Button>
        {/* <Button
            onClick={() => {
              const searchParams = new URLSearchParams(window.location.search);
              const viewpath = searchParams.get("page");
              window.open("?page=" + (viewpath ? viewpath : "home"), "_blank");
            }}
          >
            🖥Extended View
          </Button> */}
      </Toolbar>
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
      </Toolbar>
    </Box>
  );
}

