import * as React from "react";
import Box from "@mui/material/Box";
import { processEvents, convertToDecimalFormat } from "./processTokenTransfers.js"
import "./BalanceOverview.css";

function makeBalanceChangeTable(balanceChange, tokenList) {
    const rows = [];
    var prevAccount = "";
    for (const [account, tokenBalance] of Object.entries(balanceChange)) {
        for (const [token, balance] of Object.entries(tokenBalance)) {
            var addressOrSymbol = tokenList[token.toLocaleLowerCase()] ? tokenList[token.toLocaleLowerCase()].symbol : token;
            var balanceStr = tokenList[token.toLocaleLowerCase()] ? convertToDecimalFormat(balance, tokenList[token.toLocaleLowerCase()].decimals) : balance.toString();
            rows.push(
                <tr>
                    {account === prevAccount ? null : <td rowspan={Object.keys(tokenBalance).length}>{account}</td>}
                    <td>{addressOrSymbol}</td>
                    <td>{balanceStr}</td>
                </tr>
            )
            prevAccount = account;
        }
    }
    return rows;
}

function Table(props) {
    const { columnNames, events, columnDictKeys, tokenList } = props;
    return (
        <table class="table">
            <thead>
                <tr>
                    {columnNames.map((name) => <th>{name}</th>)}
                </tr>
            </thead>
            <tbody>
                {events ? events.map((event) => {
                    return (
                        <tr>
                            {
                                columnDictKeys.map((key) => {
                                    var tokenInfo = tokenList[event["token"].toLocaleLowerCase()];
                                    switch (key) {
                                        case "token":
                                            var addressOrSymbol = tokenInfo ? tokenInfo.symbol : event[key];
                                            return <td>{addressOrSymbol}</td>
                                        case "amount":
                                            if (event[key] === "infinite") {
                                                return <td>{event[key]}</td>
                                            }
                                            var balanceStr = tokenInfo ? convertToDecimalFormat(event[key], tokenInfo.decimals) : event[key].toString();
                                            return <td>{balanceStr}</td>
                                        default:
                                            return <td>{event[key]}</td>
                                    }
                                })
                            }
                        </tr>
                    )
                }) : null}
            </tbody>
        </table>
    )
}

export default function BalanceOverview(props) {
    const events = processEvents(props.events);
    return (
        <Box sx={{ width: "100%" }}>
            <div class="transfers">
                <h3>Balance Changes: </h3>
                <table class="table">
                    <thead>
                        <tr>
                            <th scope="col">Account</th>
                            <th scope="col">Token</th>
                            <th scope="col">Balance</th>
                        </tr>
                    </thead>
                    <tbody>
                        {events.userTokenBalance ? makeBalanceChangeTable(events.userTokenBalance, props.tokenList) : null}
                    </tbody>
                </table>
            </div>
            <div class="transfers">
                <h3>Token Transfers: </h3>
                <Table
                    tokenList={props.tokenList}
                    columnNames={["Sender", "Token", "Amount", "Receiver"]}
                    events={events.transferERC20Events}
                    columnDictKeys={["from", "token", "amount", "to"]}
                />
            </div>
            <div class="approvals">
                <h3>Token Approvals: </h3>
                <Table
                    tokenList={props.tokenList}
                    columnNames={["Token", "Owner", "Spender", "Amount"]}
                    events={events.approvalERC20Events}
                    columnDictKeys={["token", "owner", "spender", "amount"]}
                />
            </div>
            <div class="approvals">
                <h3>[NFT] Set Approval For All: ⚠️⚠️</h3>
                <Table
                    tokenList={props.tokenList}
                    columnNames={["Token", "Owner", "Operator", "Approved"]}
                    events={events.approvalForAllEvents}
                    columnDictKeys={["token", "owner", "operator", "approved"]}
                />
            </div>
            <div class="transfers">
                <h3>[NFT] Transfers: </h3>
                <Table
                    tokenList={props.tokenList}
                    columnNames={["Sender", "Token", "ID", "Receiver"]}
                    events={events.transferERC721Events}
                    columnDictKeys={["from", "token", "tokenID", "to"]}
                />
            </div>
            <div class="approvals">
                <h3>[NFT] Approvals: </h3>
                <Table
                    tokenList={props.tokenList}
                    columnNames={["Token", "Owner", "Spender", "ID"]}
                    events={events.approvalERC721Events}
                    columnDictKeys={["token", "owner", "spender", "tokenID"]}
                />
            </div>
        </Box>
    );
}
