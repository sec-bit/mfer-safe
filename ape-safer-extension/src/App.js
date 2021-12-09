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
  }
  return page;

  // return (
  //   <div className="App">
  //     <h1>Keep Degen And Ape In!</h1>
  //     <Routes>
  //       <Route path="/" element={<Home />} />
  //       <Route path="/txs" element={<TxDataOverview />} />
  //       <Route path="/trace/:txhash" element={<TraceView />} />
  //     </Routes>
  //   </div>
  // );
}

// App.js
function Home() {
  return (
    <React.StrictMode>
      <ButtonToolbar />
      <SimulateView />
      <TxDataTable />
      <SettingsView />
    </React.StrictMode>
  );
}

export default App;
