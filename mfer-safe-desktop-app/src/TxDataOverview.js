import { React, useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { docall } from "./utils.js";
import Box from "@mui/material/Box";
import { DataGrid } from "@mui/x-data-grid";
import functionSignatures from "./functionSignatures.json";
import BalanceOverview from "./BalanceOverview.js";
import { loadTokenList } from "./processTokenTransfers.js"


const columns = [
  { field: "id", headerName: "Index", width: 60 },
  {
    field: "pseudoTxHash",
    headerName: "Pseudo Tx Hash",
    width: 300,
    renderCell: function (params) {
      return <Link to={"/trace/" + params.value}>{params.value}</Link>;
    },
  },
  { field: "method", headerName: "Method (Guessed)", width: 200 },
  { field: "selector", headerName: "Selector", width: 100 },
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
    selector:txdata.calldata.slice(0,10),
    method: abiDict[txdata.calldata.slice(2,10)],
    from: txdata.from,
    to: txdata.to,
    execResult: txdata.execResult,
  }));
  return rows;
};

const updateTxList = function (setRows, setEvents) {
  docall("mfer_getTxs", [])
    .then((res) => res.json())
    .then(
      (result) => {
        var rows = genRows(result.result, functionSignatures);
        setRows(rows);
      },
      (error) => {
        console.log(error);
      }
    );
    docall("eth_getLogs", [{"blockhash":"0x445fd31bd4f3b47d5248cda491b447af5a0c4ed91b6dcd83ba0fb5e69526876d"}])
    .then((res) => res.json())
    .then(
      (result) => {
        setEvents(result.result);
      },
      (error) => {
        console.log(error);
      }
    );
};

export default function TxDataOverview() {
  const [rows, setRows] = useState([]);
  const [events, setEvents] = useState([]);
  const [tokenList, setTokenList] = useState({});
  useEffect(() => {
    updateTxList(setRows, setEvents);
    const interval = setInterval(() => {
      updateTxList(setRows,setEvents);
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    setTokenList(loadTokenList());
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
      <BalanceOverview events={events} tokenList={tokenList}/>
    </Box>
  );
}
