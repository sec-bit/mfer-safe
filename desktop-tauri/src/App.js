import "./App.css";
import * as React from "react";

import NavTabs from "./NavTabs";
import SimulateView from "./SimulateView";
import LogsView from "./LogsView";
import SettingsView from "./SettingsView";
import TxDataOverview from "./TxDataOverview";
import TraceView from "./TraceView";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";

function App() {
  return (
    <Router>
      <div>
        <NavTabs />
        <Routes>
          <Route exact path="/" element={<SettingsView />} />
          <Route path="/txs" element={<TxDataOverview />} />
          <Route path="/trace/:txHash" element={<TraceView />} />
          <Route path="/safemultisend" element={<SimulateView />} />
          <Route path="/logs" element={<LogsView/>} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;
