import { React, useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { docall } from "./utils.js";
import Box from "@mui/material/Box";
import { DataGrid } from "@mui/x-data-grid";

const columns = [
  { field: "id", headerName: "Index", width: 70 },

  {
    field: "pseudoTxHash",
    headerName: "Txn Hash",
    width: 350,
    renderCell: function (params) {
      return <Link to={"/trace/" + params.value}>{params.value}</Link>;
    },
  },
  { field: "method", headerName: "Method", width: 200 },
  { field: "from", headerName: "From", width: 300 },
  { field: "to", headerName: "To", width: 300 },
  { field: "execResult", headerName: "Result", width: 500 },
];

const genRows = function (txs, abiDict) {
  // debugger;
  if (txs.length === 0) {
    return [];
  }
  var rows = txs.map((txdata) => ({
    id: txdata.idx,
    pseudoTxHash: txdata.pseudoTxHash,
    method: abiDict[txdata.calldata],
    from: txdata.from,
    to: txdata.to,
    execResult: txdata.execResult,
  }));
  return rows;
};

const updateTxList = function (setRows) {
  docall("ape_getTxs", [])
    .then((res) => res.json())
    .then(
      (result) => {
        setRows(genRows(result.result, {}));
      },
      (error) => {
        console.log(error);
      }
    );
};

export default function TxDataOverview() {
  const [rows, setRows] = useState([]);
  useEffect(() => {
    updateTxList(setRows);
    const interval = setInterval(() => {
      updateTxList(setRows);
    }, 1000);
    return () => clearInterval(interval);
  }, []);
  return (
    <Box>
      <DataGrid
      autoHeight={true}
        rows={rows}
        columns={columns}
        pageSize={50}
        rowsPerPageOptions={[50]}
        checkboxSelection
      />
    </Box>
  );
}
