import React from "react";
import Link from "@mui/material/Link";
import fourByte from "4byte";
import { ethers } from "ethers";
import { docall } from "./utils.js";

import { DataGrid } from "@mui/x-data-grid";

const columns = [
  { field: "id", headerName: "Index", width: 70 },

  {
    field: "pseudoTxHash",
    headerName: "Txn Hash",
    width: 350,
    renderCell: function (params) {
      return (
        <Link href={"?page=trace&txhash=" + params.value}>{params.value}</Link>
      );
    },
  },
  { field: "method", headerName: "Method", width: 200 },
  { field: "from", headerName: "From", width: 300 },
  { field: "to", headerName: "To", width: 300 },
];

class TxDataOverview extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      txs: [],
      isLoaded: false,
      txsCache: "",
      abi: {},
      open: false,
    };
    this.updateTxList();
  }

  updateTxList() {
    docall("ape_getTxs", [])
      .then((res) => res.json())
      .then(
        (result) => {
          if (JSON.stringify(result.result) !== this.state.txsCache) {
            let txs = result.result;
            txs.map((txdata) => {
              let selector = ethers.utils.hexDataSlice(txdata.calldata, 0, 4);
              fourByte(selector).then((abis) => {
                let key = txdata.calldata;
                console.log(abis);
                abis.map((abi) => {
                  let abiBracketsTrimmed = abi
                    .replace(/[()]/gi, " ")
                    .split(" ");
                  let funcName = abiBracketsTrimmed[0];
                  console.log("funcName:", funcName);
                  // debugger;
                  let inputParameters = abiBracketsTrimmed[1].split(",");
                  try {
                    let decoded = ethers.utils.defaultAbiCoder.decode(
                      inputParameters,
                      ethers.utils.hexDataSlice(key, 4)
                    );
                    console.log(key, abi, decoded);
                    let abiDict = this.state.abi;
                    abiDict[key] = JSON.stringify(funcName, decoded);
                    this.setState({
                      abi: abiDict,
                    });
                  } catch (e) {
                    console.log(e);
                  }
                });
              });
            });
            this.setState({
              isLoaded: true,
              txs: txs,
              txsCache: JSON.stringify(txs),
            });
          } else {
            console.log("txs no change");
          }
        },
        (error) => {
          this.setState({
            isLoaded: false,
            error,
          });
        }
      );
  }

  componentDidMount() {
    this.timerID = setInterval(() => this.tick(), 1000);
  }

  componentWillUnmount() {
    clearInterval(this.timerID);
  }

  tick() {
    this.updateTxList();
  }

  render() {
    var txs = this.state.txs;
    if (txs.length === 0) {
      return <div>No txs yet</div>;
    }
    var rows = txs.map(
      (txdata) => ({
        id: txdata.idx,
        pseudoTxHash: txdata.pseudoTxHash,
        method: this.state.abi[txdata.calldata],
        from: txdata.from,
        to: txdata.to,
      })
      // <li key={number.toString()}>
      //   {number}
      // </li>
    );

    return (
      <div style={{ height: 400, width: "100%" }}>
        <DataGrid
          rows={rows}
          columns={columns}
          pageSize={50}
          rowsPerPageOptions={[50]}
          checkboxSelection
        />
      </div>
    );
  }
}

export default TxDataOverview;
