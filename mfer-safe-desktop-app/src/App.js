import "./App.css";
import * as React from "react";
import {useState, useEffect} from "react"

import NavTabs from "./NavTabs";
import SimulateView from "./SimulateView";
import LogsView from "./LogsView";
import SettingsView from "./SettingsView";
import TxDataOverview from "./TxDataOverview";
import TraceView from "./TraceView";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { listen } from "@tauri-apps/api/event";

function App() {
  const [log, setLog] = useState("")
  useEffect(() => {
    console.log("init tauri event listener")
    listen("mfernode-event", (event) => {
      if (event.payload !== undefined) {
        setLog((log) => {
          var logLines = log.split("\n").slice(0, 500);
          log = logLines.join("\n");
          return event.payload + "\n" + log
        });
      }
    });
  }, []);
  return (
    <Router>
      <div>
        <NavTabs />
        <Routes>
          <Route exact path="/" element={<SettingsView />} />
          <Route path="/txs" element={<TxDataOverview />} />
          <Route path="/trace/:txHash" element={<TraceView />} />
          <Route path="/safemultisend" element={<SimulateView />} />
          <Route path="/logs" element={<LogsView log={log} />} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;
