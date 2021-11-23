import React from "react";
import ReactDOM from "react-dom";
import "./index.css";
import ButtonToolbar from "./ButtonToolbar";
import TxData from "./TxDataTable";
import SimulateView from "./SimulateView";
import reportWebVitals from "./reportWebVitals";

ReactDOM.render(
  <React.StrictMode>
    <ButtonToolbar />
  </React.StrictMode>,
  document.getElementById("button-toolbar")
);

ReactDOM.render(
  <React.StrictMode>
    <TxData />
  </React.StrictMode>,
  document.getElementById("txdata")
);

ReactDOM.render(
  <React.StrictMode>
    <SimulateView />
  </React.StrictMode>,
  document.getElementById("simulation-view")
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
