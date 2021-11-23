import React from "react";
import "./TxDataTable.css";
import fourByte from "4byte";
import { ethers } from "ethers";
import Collapse from "@mui/material/Collapse";
import IconButton from "@mui/material/IconButton";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import Paper from "@mui/material/Paper";
import KeyboardArrowDownIcon from "@mui/icons-material/KeyboardArrowDown";
import KeyboardArrowUpIcon from "@mui/icons-material/KeyboardArrowUp";

class TxData extends React.Component {
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

  docall(cmd, params) {
    var body = {
      jsonrpc: "2.0",
      id: 123,
      method: cmd,
      params: params,
    };
    var ret = fetch("http://127.0.0.1:10545/", {
      headers: {
        accept: "*/*",
        "content-type": "application/json",
      },
      referrerPolicy: "strict-origin-when-cross-origin",
      body: JSON.stringify(body),
      method: "POST",
      mode: "cors",
      credentials: "omit",
    });
    return ret;
  }

  updateTxList() {
    this.docall("ape_getTxs", [])
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
            console.log("txs not changes");
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
      (txdata) => (
        // <tr key={txdata.idx} className="TxData-row-tr">
        //   <td className="TxData-td">{txdata.to}</td>
        //   <td className="TxData-td">{this.state.abi[txdata.calldata]}</td>
        //   <td className="TxData-td">{txdata.execResult}</td>
        //   <td className="TxData-td">{txdata.calldata}</td>
        // </tr>
        <React.Fragment>
          <TableRow>
            <TableCell>
              <IconButton
                aria-label="expand row"
                size="small"
                onClick={() => this.setState({ open: !this.state.open })}
              >
                {this.state.open ? (
                  <KeyboardArrowUpIcon />
                ) : (
                  <KeyboardArrowDownIcon />
                )}
              </IconButton>
            </TableCell>
            <TableCell component="th" scope="row">
              {txdata.idx}
            </TableCell>
            <TableCell>{txdata.to}</TableCell>
            <TableCell>{this.state.abi[txdata.calldata]}</TableCell>
            <TableCell>{txdata.execResult}</TableCell>
          </TableRow>
          <TableRow>
            <TableCell>
              <Collapse in={this.state.open} timeout="auto" unmountOnExit>
                {txdata.calldata}
              </Collapse>
            </TableCell>
          </TableRow>
        </React.Fragment>
      )
      // <li key={number.toString()}>
      //   {number}
      // </li>
    );

    return (
      // <table className="TxData-table">
      //   <tbody>
      //     <tr className="TxData-tr">
      //       <th className="TxData-th">To</th>
      //       <th className="TxData-tr">ABI</th>
      //       <th className="TxData-tr">Execution Result</th>
      //       <th className="TxData-tr">Calldata</th>
      //     </tr>
      rows
      // </tbody>
      // </table>
    );
  }
}

export default TxData;
