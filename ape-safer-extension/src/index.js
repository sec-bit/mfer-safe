import React from "react";
import ReactDOM from "react-dom";
import "./index.css";
// import { BrowserRouter } from "react-router-dom";
// import ButtonToolbar from "./ButtonToolbar";
// import TxDataTable from "./TxDataTable";
// import SimulateView from "./SimulateView";
// import SettingsView from "./SettingsView";
import App from "./App";
import reportWebVitals from "./reportWebVitals";

ReactDOM.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
  document.getElementById("root")
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
