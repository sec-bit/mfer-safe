import "./App.css";
import * as React from "react";
import { useState, useEffect } from "react";

// import { Routes, Route } from "react-router-dom";
import ButtonToolbar from "./ButtonToolbar";
import TxDataTable from "./TxDataTable";
import SimulateView from "./SimulateView";
import SettingsView from "./SettingsView";
import TxDataOverview from "./TxDataOverview";
import TraceView from "./TraceView";
import DebugView from "./DebugView";
import NavigationBar from "./NavigationBar";

function App() {
  const [path, setPath] = useState({});

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const viewpath = searchParams.get("page");
    setPath(viewpath ? viewpath : "home");
  });
  let page = null;
  switch (path) {
    case "home":
      page = <Home />;
      break;
    case "txs":
      page = <TxDataOverview />;
      break;
    case "trace":
      page = <TraceView />;
      break;
    case "debug":
      page = <DebugView />;
      break;
    case "safemultisend":
      page = <SimulateView />;
      break;
    case "navigationbar":
      return (
        <React.StrictMode>
          <NavigationBar />
        </React.StrictMode>
      );
  }
  return (
    <React.StrictMode>
      <ButtonToolbar />
      {page}
    </React.StrictMode>
  );
}

// App.js
function Home() {
  return (
    <React.StrictMode>
      <SettingsView />
      <TxDataTable />
    </React.StrictMode>
  );
}

export default App;
